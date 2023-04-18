package pluginmgr_test

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go/src/lib/lomcommon"
	//"go/src/lib/lomipc"
	//"go/src/pluginmgr/pluginmgr_common"
	//"go/src/plugins/plugins_common"
	//"go/src/plugins/plugins_files"
	//"io/ioutil"
	//"log/syslog"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"testing"
	"time"
	//"encoding/json"
	//"errors"
	"io/ioutil"
)

//----------------------------------------- Test GoroutineTracker ---------------------------------------- //

type mockFunction struct {
	mock.Mock
}

// some dummy function to be called by the goroutine
func (m *mockFunction) exec(vv ...interface{}) {
	var timeval time.Duration = time.Duration(0) * time.Millisecond
	var wg *sync.WaitGroup = nil
	for _, v := range vv {
		switch val := v.(type) {
		case int:
			timeval = time.Duration(val) * time.Millisecond
		case *sync.WaitGroup:
			wg = val
		default:
			fmt.Println("Value is of an unknown type")
		}
	}
	time.Sleep(timeval)
	m.Called()

	if wg != nil {
		wg.Done()
	}
}

	func TestGoroutineTracker(t *testing.T) {
		mygoroutinetracker := lomcommon.NewGoroutineTracker()
		wg := sync.WaitGroup{}

		t.Run("Test - Start and Wait", func(t *testing.T) {
			//create mock object
			mockFunc := &mockFunction{}

			// set expectation
			mockFunc.On("exec").Once()

			// call mockFunc.exec in goroutine, name of goroutine is "test_goroutine" & function called is "exec"
			mygoroutinetracker.Start("test_goroutine", mockFunc.exec, 1000)

			// Check that the goroutine is running
			time.Sleep(10 * time.Millisecond) // Wait for the goroutine to start
			assert.True(t, mygoroutinetracker.IsRunning("test_goroutine"))

			// Check that the goroutine is no longer running after waiting for it
			mygoroutinetracker.Wait("test_goroutine")
			assert.False(t, mygoroutinetracker.IsRunning("test_goroutine"))

			// Check that the function exec() was called as per previous expectation
			mockFunc.AssertExpectations(t)
		})

		t.Run("Test - Start goroutine with existing name", func(t *testing.T) {
			mockFunc := &mockFunction{}
			mockFunc.On("exec").Once()
			mygoroutinetracker.Start("test_goroutine2", mockFunc.exec, 1000)

			// Attempting to start a goroutine with the same name
			assert.PanicsWithValue(t, "Cannot start goroutine. Name \"test_goroutine2\" already exists", func() {
				mygoroutinetracker.Start("test_goroutine2", mockFunc.exec)
			})

			mygoroutinetracker.Wait("test_goroutine2")

			assert.False(t, mygoroutinetracker.IsRunning("test_goroutine2"))

			mockFunc.AssertExpectations(t)
		})

		t.Run("Test - List goroutines statistics", func(t *testing.T) {
			mockFunc1 := &mockFunction{}
			mockFunc2 := &mockFunction{}

			// Wait for both goroutines to complete
			wg.Add(2)

			// Start two goroutines with different names
			mockFunc1.On("exec").Once()
			mygoroutinetracker.Start("test_goroutine3", mockFunc1.exec, 1000, &wg)

			mockFunc2.On("exec").Once()
			mygoroutinetracker.Start("test_goroutine4", mockFunc2.exec, 1000, &wg)

			// Check that both goroutines are running
			time.Sleep(10 * time.Millisecond) // Wait for the goroutines to start
			assert.True(t, mygoroutinetracker.IsRunning("test_goroutine3"))
			assert.True(t, mygoroutinetracker.IsRunning("test_goroutine4"))

			// Check that List returns both names
			infos := mygoroutinetracker.InfoList()
			var names []string
			for _, info := range infos {
				if gi, ok := info.(lomcommon.GoroutineInfo); ok {
					if gi.Status == lomcommon.GoroutineStatusRunning {
						names = append(names, gi.Name)
					}
				}
			}
			assert.ElementsMatch(t, []string{"test_goroutine3", "test_goroutine4"}, names)

			// Check that both goroutines are no longer running after waiting for them
			wg.Wait()
			assert.False(t, mygoroutinetracker.IsRunning("test_goroutine3"))
			assert.False(t, mygoroutinetracker.IsRunning("test_goroutine4"))

			mockFunc2.AssertExpectations(t)
			mockFunc1.AssertExpectations(t)
		})

		t.Run("Test - STart Time", func(t *testing.T) {
			mockFunc := &mockFunction{}
			mockFunc.On("exec").Once()

			wg.Add(1)
			mygoroutinetracker.Start("test_goroutine5", mockFunc.exec, 1000, &wg)

			funcgetstartedtime := func() string {
				infos := mygoroutinetracker.InfoList()
				for _, info := range infos {
					if gi, ok := info.(lomcommon.GoroutineInfo); ok {
						if gi.Name == "test_goroutine5" {
							return gi.StartTime.String()
						}
					}
				}
				return ""
			}

			// Check that the goroutine is running
			time.Sleep(10 * time.Millisecond) // Wait for the goroutine to start
			assert.True(t, mygoroutinetracker.IsRunning("test_goroutine5"))

			gottime, _ := mygoroutinetracker.GetTimeStarted("test_goroutine5")
			assert.Equal(t, funcgetstartedtime(), gottime)

			// Check that the goroutine is no longer running after waiting for it
			wg.Wait()
			assert.False(t, mygoroutinetracker.IsRunning("test_goroutine5"))

			// check the time must be ""
			gottime, _ = mygoroutinetracker.GetTimeStarted("test_goroutine5")
			assert.Equal(t, "", gottime)

			// Now get the time for unknown goroutine
			gottime, _ = mygoroutinetracker.GetTimeStarted("dummy_goroutine")
			assert.Equal(t, "", gottime)

			mockFunc.AssertExpectations(t)
		})

}

type MyStruct struct {
	Name string
}

func (s *MyStruct) Print() {
	fmt.Println(s.Name)
}

func (s *MyStruct) PrintArg(a int, b int) {
	fmt.Printf("Testing with args .......... %d %d", a,b)
}

func TestGoroutine2(t *testing.T) {

	// Create Goroutine Tracker which will be used to track all goroutines in the process
	goroutinetracker := lomcommon.NewGoroutineTracker()
	if goroutinetracker == nil {
		panic("Error creating goroutine tracker")
	}
	//ms := &MyStruct{"hello"}
	goroutinetracker.Start("test", func (a int, b int) { fmt.Printf("1111111111111111 %d %d", a,b) }, 10, 20)
	goroutinetracker.Start("test1", func () { fmt.Printf("1111111111111111") })
	var ptr = &MyStruct{"hello"}
	goroutinetracker.Start("test3", ptr.Print)
	goroutinetracker.Start("test4", ptr.PrintArg, 10, 20)

	//panic calls
	//goroutinetracker.Start("test5", func (a int, b int) { fmt.Printf("1111111111111111 %d %d", a,b) })
	//goroutinetracker.Start("test6", func () { fmt.Printf("1111111111111111") }, 10, 20)
	//goroutinetracker.Start("test7", ptr.Print, 10, 20)
	//goroutinetracker.Start("test8", ptr.PrintArg)

}
//------------------------------------------ End of Test GoroutineTracker ------------------------------------------//

//------------------------------------------ config.go -------------------------------------------------------------//

type mockReadFile struct {
	mock.Mock
}

func (m *mockReadFile) ReadFile(filename string) ([]byte, error) {
	args := m.Called(filename)
	return args.Get(0).([]byte), args.Error(1)
}

func TestReadProcsConf(t *testing.T) {
	testData_error := []byte(`{
		"proc_0": {
			"link_crc": {
				"name": "link_crc",
				"version": "00.01.1",
				"path": " /path/"
			},
			"link_flap"d: {
				"name": "link_flap",
				"version": "02.00.1",
				"path": " /path/"
			}
		},
		"proc_1": {
			"bgp_holddown": {
				"name": "bgp_holddown",
				"version": "02_1",
				"path": " /path/"
			}
		}
	}`)

	t.Run("success", func(t *testing.T) {
		configFiles := &lomcommon.ConfigFiles_t{}
		configFiles.ProcsFl = "./proc_conf.json"
		configMgr, err := lomcommon.InitConfigMgr(configFiles)
		if err != nil {
			t.Errorf("Error in InitConfigMgr: %v", err)
		}

		assert.Nil(t, err)
		assert.Equal(t, "link_crc", configMgr.ProcsConfig["link_crc"].Name)
		assert.Equal(t, "02.00.1", configMgr.ProcsConfig["link_flap"].Version)
	})

	t.Run("readFile_error", func(t *testing.T) {
		configFiles := &lomcommon.ConfigFiles_t{}
		configFiles.ProcsFl = "./proc_conf_dummy.json"
		_, err := lomcommon.InitConfigMgr(configFiles)

		assert.NotNil(t, err)
		//assert.EqualError(t, err, "error reading file")
		assert.Regexp(t, regexp.MustCompile("Procs: ./proc_conf_dummy.json: open ./proc_conf_dummy.json: no such file or directory"), err.Error())
	})

	t.Run("unmarshal_error", func(t *testing.T) {
		// create the file
		err := ioutil.WriteFile("/tmp/test.json", testData_error, 0644)
		if err != nil {
			panic(err)
		}

		configFiles := &lomcommon.ConfigFiles_t{}
		configFiles.ProcsFl = "/tmp/test.json"
		_, err = lomcommon.InitConfigMgr(configFiles)

		assert.NotNil(t, err)
		//assert.EqualError(t, err, "invalid character 'i' looking for beginning of value")
		assert.Regexp(t, regexp.MustCompile("Procs: /tmp/test.json: invalid character '.+' after object key"), err.Error())

		// delete the file
		err = os.Remove("/tmp/test.json")
		if err != nil {
			panic(err)
		}
	})

	t.Run("proc keys invalid", func(t *testing.T) {
		configFiles := &lomcommon.ConfigFiles_t{}
		configFiles.ProcsFl = "./proc_conf.json"
		configMgr, err := lomcommon.InitConfigMgr(configFiles)

		assert.Nil(t, err)
		assert.NotEqual(t, "link_crc", configMgr.ProcsConfig["link_crc_c"].Name)
		assert.NotEqual(t, "dummy_path", configMgr.ProcsConfig["link_crc"].Path)
	})

	t.Run("proc ID invalid", func(t *testing.T) {
		configFiles := &lomcommon.ConfigFiles_t{}
		configFiles.ProcsFl = "./proc_conf.json"
		lomcommon.ProcID = "proc_3"
		_, err := lomcommon.InitConfigMgr(configFiles)

		assert.NotNil(t, err)
		assert.Regexp(t, regexp.MustCompile(`.*Failed to get config for proc \(\w+\)`), err.Error())
	})
}

func TestValidateConfigFile(t *testing.T) {
	location := "/path/to/config"
	filename := "config.json"

	// Test when absolute path cannot be created
	_, err := lomcommon.ValidateConfigFile("invalid_path", filename)
	assert.Error(t, err)
	assert.Regexp(t, regexp.MustCompile(`config file.*config.json.*does not exist.*`), err.Error())

	// Test when config file does not exist
	_, err = lomcommon.ValidateConfigFile(location, filename)
	assert.Error(t, err)

	// Create temporary config file for testing
	testData := []byte(`{"foo":"bar"}`)
	tmpFile, err := ioutil.TempFile("", "test-config_gg-*.json")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	_, err = tmpFile.Write(testData)
	assert.NoError(t, err)

	// Test when config file exists
	configFileAbs, err := lomcommon.ValidateConfigFile(filepath.Dir(tmpFile.Name()), filepath.Base(tmpFile.Name()))
	assert.NoError(t, err)
	assert.Equal(t, tmpFile.Name(), configFileAbs)

	// Test an directory
	_, err = lomcommon.ValidateConfigFile("/tmp", "")
	assert.Error(t, err)
}

//------------------------------------------ End config.go -------------------------------------------------------------//

//------------------------------------------ REad Env variables test(helpers.go) -------------------------------------------------------------//

func TestGetEnvVarString(t *testing.T) {
	sessionId, exists := os.LookupEnv("XDG_SESSION_ID")
	if !exists {
		panic("Session ID not found")
	}

	// Call the LoadEnvironemntVariables function
	lomcommon.LoadEnvironemntVariables()

	// Call the GetEnvVarString function
	value, exists := lomcommon.GetEnvVarString("ENV_session_id")

	// Assert that the value and exists variables are correct
	assert.Equal(t, sessionId, value)
	assert.True(t, exists)

	// Call the GetEnvVarString function with a non-existent key
	value, exists = lomcommon.GetEnvVarString("non_existent_key")

	// Assert that the value and exists variables are correct
	assert.Equal(t, "", value)
	assert.False(t, exists)

	// Call the GetEnvVarString function with a key that has no corresponding value in the environment
	value, exists = lomcommon.GetEnvVarString("ENV_lom_conf_location")

	// Assert that the value and exists variables are correct
	assert.Equal(t, "", value)
	assert.True(t, exists)
}

func TestGetEnvVarInteger(t *testing.T) {
	sessionId, exists := os.LookupEnv("XDG_SESSION_ID")
	if !exists {
		panic("Session ID not found")
	}
	sessionIdInt, ok := strconv.Atoi(sessionId)
	if ok != nil {
		panic("Session ID integer conversion failed")
	}

	lomcommon.LoadEnvironemntVariables()

	value, exists := lomcommon.GetEnvVarInteger("ENV_session_id") // in int

	assert.Equal(t, sessionIdInt, value)
	assert.True(t, exists)

	// Call the GetEnvVarInteger function with a non-existent key
	value, exists = lomcommon.GetEnvVarInteger("non_existent_key")

	// Assert that the value and exists variables are correct
	assert.Equal(t, 0, value)
	assert.False(t, exists)

	// Call the GetEnvVarInteger function with a key that has no corresponding value in the environment
	value, exists = lomcommon.GetEnvVarInteger("ENV_lom_conf_location")

	// Assert that the value and exists variables are correct
	assert.Equal(t, 0, value)
	assert.False(t, exists)
}

func TestGetEnvVarFloat(t *testing.T) {
	sessionId, exists := os.LookupEnv("XDG_SESSION_ID")
	if !exists {
		panic("Session ID not found")
	}
	sessionIdFlt, ok := strconv.ParseFloat(sessionId, 64)
	if ok != nil {
		panic("Session ID float conversion failed")
	}

	lomcommon.LoadEnvironemntVariables()

	value, exists := lomcommon.GetEnvVarFloat("ENV_session_id")

	assert.Equal(t, sessionIdFlt, value)
	assert.True(t, exists)

	// Call the GetEnvVarFloat function with a non-existent key
	value, exists = lomcommon.GetEnvVarFloat("non_existent_key")

	// Assert that the value and exists variables are correct
	assert.Equal(t, 0.0, value)
	assert.False(t, exists)

	// Call the GetEnvVarFloat function with a key that has no corresponding value in the environment
	value, exists = lomcommon.GetEnvVarFloat("ENV_lom_conf_location")

	// Assert that the value and exists variables are correct
	assert.Equal(t, 0.0, value)
	assert.False(t, exists)
}

func TestGetEnvVaAny(t *testing.T) {
	sessionId, exists := os.LookupEnv("XDG_SESSION_ID")
	if !exists {
		panic("Session ID not found")
	}
	lomcommon.LoadEnvironemntVariables()

	value, exists := lomcommon.GetEnvVarAny("ENV_session_id")

	assert.Equal(t, sessionId, value)
	assert.True(t, exists)

	// Call the GetEnvVarAny function with a non-existent key
	value, exists = lomcommon.GetEnvVarAny("non_existent_key") // Interface comparision

	// Assert that the value and exists variables are correct
	assert.Equal(t, "", value)
	assert.False(t, exists)

	// Call the GetEnvVarAny function with a key that has no corresponding value in the environment
	value, exists = lomcommon.GetEnvVarAny("ENV_lom_conf_location")

	// Assert that the value and exists variables are correct
	assert.Equal(t, "", value)
	assert.True(t, exists)
}

func TestGetEnvVarFromOS(t *testing.T) {
	// Set an environment variable
	err := os.Setenv("MY_ENV_VAR", "LOM")
	if err != nil {
		panic(err)
	}

	// Get the value of the environment variable
	value := os.Getenv("MY_ENV_VAR")
	fmt.Println(value)

	// The environment variable exists
	envVarName := "MY_ENV_VAR"
	expectedValue := "LOM"
	actualValue, exists := lomcommon.GetEnvVarFromOS(envVarName)
	assert.True(t, exists)
	assert.Equal(t, expectedValue, actualValue)

	// Clear an environment variable
	os.Unsetenv("MY_ENV_VAR")

	// The environment variable does not exist
	envVarName = "nonexistent_env_var"
	expectedValue = ""
	actualValue, exists = lomcommon.GetEnvVarFromOS(envVarName)
	assert.False(t, exists)
	assert.Equal(t, expectedValue, actualValue)
}

//------------------------------------------ End Read Env variables test(helpers.go) -------------------------------------------------------------//
