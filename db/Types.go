package db;

import (
    "time"
)

type DBTable  interface {
    ExerciseType |
    ExerciseFocus |
    Exercise |
    Rotation |
    BodyWeight |
    TrainingLog |
    Client |
    ModelState
};

type ExerciseType struct {
    Id int;
    T string;
    Description string;
};

type ExerciseFocus struct {
    Id int;
    Focus string;
};

type Exercise struct {
    Id int;
    Name string;
    TypeID int;
    FocusID int;
};

//Start and end dates are inclusive
type Rotation struct {
    Id int;
    ClientID int;
    StartDate time.Time;
    EndDate time.Time;
};

type BodyWeight struct {
    Id int;
    ClientID int;
    Weight float32;
    Date time.Time;
};

type TrainingLog struct {
    Id int;
    ClientID int;
    ExerciseID int;
    RotationID int;
    DatePerformed time.Time;
    Weight float32;
    Sets float32;
    Reps int;
    Intensity float64;
    Effort float64;
    FatigueIndex int;
    Volume float64;
};

type Client struct {
    Id int;
    FirstName string;
    LastName string;
    Email string;
};

type ModelState struct {
    Id int;
    ClientID int;
    ExerciseID int;
    Date time.Time;
    A,B,C,D,Eps,Eps2 float64;
    TimeFrame int;
    Rcond float64;
};
