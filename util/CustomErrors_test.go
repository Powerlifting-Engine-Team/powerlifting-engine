package util;

import (
    "fmt"
    "testing"
)

func TestErrorEquality(t *testing.T){
    errs:=map[string]errorType{
        "SqlScriptNotFound": SqlScriptNotFound,
        "DataConversion": DataConversion,
        "NoKnownDataConversion": NoKnownDataConversion,
        "FilterRemovedAllColumns": FilterRemovedAllColumns,
        "DataVersionMalformed": DataVersionMalformed,
        "SettingsFileNotFound": SettingsFileNotFound,
        "MatrixDimensionsDoNotAgree": MatrixDimensionsDoNotAgree,
        "InverseOfNonSquareMatrix": InverseOfNonSquareMatrix,
        "SingularMatrix": SingularMatrix,
        "MatrixSingularToWorkingPrecision": MatrixSingularToWorkingPrecision,
        "MissingVariable": MissingVariable,
        "MalformedCSVFile": MalformedCSVFile,
        "NonStructValue": NonStructValue,
        "SliceZippingError": SliceZippingError,
    };
    isErrs:=map[string]isErrorType{
        "SqlScriptNotFound": IsSqlScriptNotFound,
        "DataConversion": IsDataConversion,
        "NoKnownDataConversion": IsNoKnownDataConversion,
        "FilterRemovedAllColumns": IsFilterRemovedAllColumns,
        "DataVersionMalformed": IsDataVersionMalformed,
        "SettingsFileNotFound": IsSettingsFileNotFound,
        "MatrixDimensionsDoNotAgree": IsMatrixDimensionsDoNotAgree,
        "InverseOfNonSquareMatrix": IsInverseOfNonSquareMatrix,
        "SingularMatrix": IsSingularMatrix,
        "MatrixSingularToWorkingPrecision": IsMatrixSingularToWorkingPrecision,
        "MissingVariable": IsMissingVariable,
        "MalformedCSVFile": IsMalformedCSVFile,
        "NonStructValue": IsNonStructValue,
        "SliceZippingError": IsSliceZippingError,
    };
    for k,_:=range(errs){
        iterErr:=errs[k]("testAddendum");
        if !isErrs[k](iterErr) {
            t.Errorf(fmt.Sprintf("%s is returning false negative.",k));
        }
        if isErrs[k](DataVersionNotAvailable) {
            t.Errorf(fmt.Sprintf("%s is returning false positive.",k));
        }
    }
}
