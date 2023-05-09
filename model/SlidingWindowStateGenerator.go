package model;

import (
    "fmt"
    "time"
    stdMath "math"
    "github.com/barbell-math/block/db"
    "github.com/barbell-math/block/util/algo"
    "github.com/barbell-math/block/util/algo/iter"
    "github.com/barbell-math/block/util/dataStruct"
    mathUtil "github.com/barbell-math/block/util/math"
)

type SlidingWindowStateGen struct {
    allotedThreads int;
    windowLimits dataStruct.Pair[int,int];
    timeFrameLimits dataStruct.Pair[int,int];
    windowValues [][]dataPoint;
    optimalMs db.ModelState;
    lr mathUtil.LinearReg[float64];
    withinWindowLimits (func(t time.Time) bool);
};

//The lr and window values are not created until the generate prediction method
//in order to make sure the slice values are unique. (ie. It makes sure multiple
//threads don't access the same slices because they are not deep copied even
//though the method does not have a pointer receiver.)
func NewSlidingWindowStateGen(
        timeFrameLimits dataStruct.Pair[int,int],
        windowLimits dataStruct.Pair[int,int],
        allotedThreads int) (SlidingWindowStateGen,error) {
    rv:=SlidingWindowStateGen{
        allotedThreads: mathUtil.Constrain(allotedThreads,1,stdMath.MaxInt),
        timeFrameLimits: dataStruct.Pair[int,int]{
            A: -mathUtil.Abs(timeFrameLimits.A),
            B: -mathUtil.Abs(timeFrameLimits.B),
        }, windowLimits: dataStruct.Pair[int,int]{
            A: -mathUtil.Abs(windowLimits.A),
            B: -mathUtil.Abs(windowLimits.B),
        }, optimalMs: db.ModelState{
            Mse: stdMath.Inf(1),
        },
    };
    if rv.timeFrameLimits.A<rv.timeFrameLimits.B {
        return rv,InvalidPredictionState("min time frame > max time frame");
    } else if rv.windowLimits.A<rv.windowLimits.B {
        return rv,InvalidPredictionState("min window >= max window, should be <");
    } else if rv.windowLimits.B<rv.timeFrameLimits.B {
        return rv,InvalidPredictionState("max window >= max time frame, should be <");
    } else if rv.windowLimits.A>rv.timeFrameLimits.A {
        return rv,InvalidPredictionState("min window <= min time frame, should be >");
    }
    return rv,nil;
}

func (s SlidingWindowStateGen)GenerateClientModelStates(
        d *db.DB,
        c db.Client,
        ch chan<- []error){
    stateGenType,err:=db.GetStateGeneratorByName(d,"Sliding Window");
    err=db.CustomReadQuery(d,missingModelStatesForGivenStateGenQuery(),[]any{
        c.Id,stateGenType.Id,
    }, func (m *missingModelStateData) bool {
        fmt.Printf("Need ms for %+v\n",m);
        //SLIDING_WINDOW_DEBUG.Log("Need ms: %+v\n",m);
        return true;
    });
    fmt.Println(err);
}

func (s SlidingWindowStateGen)GenerateModelState(
        d *db.DB,
        missingData missingModelStateData,
        ch chan<- StateGeneratorRes) error {
    var curDate time.Time;
    s.setWithinWindowLimits(missingData.Date);
    s.lr=fatigueAwareModel();
    err:=db.CustomReadQuery(d,timeFrameQuery(),[]any{
        missingData.Date.AddDate(0, 0, s.timeFrameLimits.A),
        missingData.Date.AddDate(0, 0, s.timeFrameLimits.B),
        missingData.ExerciseID,
        missingData.ClientID,
    }, func (d *dataPoint) bool {
        if !curDate.Equal(d.DatePerformed) {
            if len(s.windowValues)>0 {
                s.calcAndSetModelState(d,&missingData);
            }
            curDate=d.DatePerformed;
        }
        if s.withinWindowLimits(curDate) {
            s.updateWindowValues(d);
        }
        s.updateLrSummations(d);    //Time frame limits guaranteed by query
        return !(curDate.Before(
            missingData.Date.AddDate(0, 0, s.windowLimits.B),
        ) && len(s.windowValues)==0);
    });
    if err==nil && len(s.windowValues)==0 {
        return NoDataInSelectedWindow(fmt.Sprintf(
            "Date: %s Min window: %d Max window: %d",
            missingData.Date, s.windowLimits.A, s.windowLimits.B,
        ));
    }
    fmt.Println(err);
    return err;
}

//Algo steps:
//  1. For each day in the window values array
//      1. Generate a prediction for every value in that day
//      2. Calculate the square error (SE) for every value in that day. If the
//         pred and actual lists are different lengths then it means there was
//         an error generating intensity predictions and the model state should
//         be dis-regarded
//      3. Assuming the previous step succeeded, calculate the total SE
//      4. Calculate the mean SE (MSE)
//      5. If the MSE is less than the current lowest MSE then save a new
//         model state
func (s *SlidingWindowStateGen)calcAndSetModelState(
        d *dataPoint,
        missingData *missingModelStateData) error {
    var numPoints float64=0;
    var cumulativeSe float64=0.0;
    res,rcond,_:=s.lr.Run();
    for _,w:=range(s.windowValues) {
        actual,_:=iter.Map(iter.SliceElems(w),
        func(index int, w dataPoint) (float64,error) {
                return w.Intensity,nil;
        }).Collect();
        pred,_:=iter.Map(iter.SliceElems(w),
            func(index int, w dataPoint) (float64,error) {
                return intensityPredFromLinReg(res,&w);
        }).Collect()
        if se,err:=mathUtil.SqErr(actual,pred); err==nil {
            seTot,_:=iter.SliceElems(se).Reduce(0,mathUtil.Add[float64]);
            cumulativeSe+=seTot;
            numPoints+=float64(len(w));
            if cumulativeSe/numPoints<s.optimalMs.Mse {
                s.saveModelState(res,rcond,cumulativeSe/numPoints,
                    daysBetween(s.windowValues[0][0].DatePerformed,
                        w[0].DatePerformed,
                    ), daysBetween(missingData.Date,d.DatePerformed),
                );
            }
        } else {
            return err;
        }
    }
    return nil;
}

func (s *SlidingWindowStateGen)saveModelState(
        res mathUtil.LinRegResult[float64],
        rcond float64,
        mse float64,
        winLen int,
        timeFrameLen int){
    s.optimalMs.Eps=stdMath.Max(res.GetConstant(0),0);
    //s.optimalMs.Eps1=res.GetConstant(1);
    s.optimalMs.Eps2=res.GetConstant(1);
    s.optimalMs.Eps3=res.GetConstant(2);
    s.optimalMs.Eps4=res.GetConstant(3);
    s.optimalMs.Eps5=stdMath.Max(res.GetConstant(4),0);
    s.optimalMs.Eps6=stdMath.Max(res.GetConstant(5),0);
    s.optimalMs.Eps7=stdMath.Max(res.GetConstant(6),0);
    s.optimalMs.TimeFrame=timeFrameLen;
    s.optimalMs.Win=winLen;
    s.optimalMs.Rcond=rcond;
    s.optimalMs.Mse=mse;
    SLIDING_WINDOW_MS_DEBUG.Log("ModelState",s.optimalMs);
}

func (s *SlidingWindowStateGen)updateWindowValues(d *dataPoint){
    l:=len(s.windowValues);
    if l==0 || !s.windowValues[l-1][0].DatePerformed.Equal(d.DatePerformed) {
        s.windowValues=append(s.windowValues,[]dataPoint{*d});
    } else {
        s.windowValues[l-1]=append(s.windowValues[l-1],*d);
    }
    SLIDING_WINDOW_DP_DEBUG.Log("WindowDataPoint",d);
}

func (s *SlidingWindowStateGen)updateLrSummations(d *dataPoint){
    s.lr.UpdateSummations(map[string]float64{
        "I": d.Intensity, "R": d.Reps, "E": d.Effort, "S": d.Sets,
        "F_w": d.InterWorkoutFatigue, "F_e": d.InterExerciseFatigue,
    });
    SLIDING_WINDOW_DP_DEBUG.Log("DataPoint",d);
}

func (s *SlidingWindowStateGen)setWithinWindowLimits(
        missingDataTime time.Time){
    fmt.Println(missingDataTime.AddDate(0 ,0, s.windowLimits.A));
    fmt.Println(missingDataTime.AddDate(0 ,0, s.windowLimits.B));
    s.withinWindowLimits=between(
        missingDataTime.AddDate(0, 0, s.windowLimits.A),
        missingDataTime.AddDate(0, 0, s.windowLimits.B),
    );
}

//this will eventually need to be moved to some common utility file
//Due to the confusing nature of time, both after and before are **inclusive**
//After is the 'future' time, before is the 'past' time
func between(after time.Time, before time.Time) (func(t time.Time) bool) {
    //if the 'past' time is after the 'future' time then switch them
    if before.After(after) {
        tmp:=before;
        before=after;
        after=tmp;
    }
    return func(t time.Time) bool {
        return (before.AddDate(0, 0, -1).Before(t) &&
            after.AddDate(0, 0, 1).After(t));
    }
}

func daysBetween(after time.Time, before time.Time) int {
    if before.After(after) {
        tmp:=before;
        before=after;
        after=tmp;
    }
    return int(after.Sub(before).Hours()/24);
}
