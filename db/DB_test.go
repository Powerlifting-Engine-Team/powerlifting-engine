package db;

import (
    "fmt"
    "database/sql"
    "testing"
    "github.com/carmichaeljr/powerlifting-engine/util"
)

var testDB CRUD;

func TestMain(m *testing.M){
    setup();
    m.Run();
    teardown();
}

func setup(){
    var err error=nil;
    testDB,err=NewCRUD("localhost",5432,"carmichaeljr","test");
    if err!=nil && err!=util.DataVersionNotAvailable {
        panic("Could not open database for testing.");
    }
    err=testDB.execSQLScript("../sql/globalInit.sql");
    if err!=nil && util.IsSqlScriptNotFound(err) {
        panic("Could not find 'globalInit.sql' file for testing.");
    }
}

func teardown(){
    testDB.execSQLScript("../sql/globalInit.sql");
    testDB.Close();
}

func TestVersion(t *testing.T){
    _,err:=testDB.getDataVersion();
    if err!=sql.ErrNoRows {
        t.Errorf(
            "Err getting version before adding it. Expected: (%s), Got: (nil)",
            sql.ErrNoRows,
        );
    }

    testDB.addDataVersion(-1);
}

func TestCreateExerciseType(t *testing.T){
    id,err:=testDB.CreateExerciseType(
        ExerciseType{
            _type: "TestType",
            description: "TestTypeDescription",
        },
    );
    if err!=nil && id!=0 {
        t.Errorf(
            "Err creating exercise type. Expected: (%d,nil), Got: (%d,%s)",
            0 ,id,fmt.Sprint(err),
        );
    }
    id,err=testDB.CreateExerciseType(
        ExerciseType{
            _type: "TestType1",
            description: "TestTypeDescription1",
        },
    );
    if err!=nil && id!=1 {
        t.Errorf(
            "Err creating exercise type. Expected: (%d,nil), Got: (%d,%s)",
            1,id,fmt.Sprint(err),
        );
    }
}
