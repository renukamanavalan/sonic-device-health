package plugins_common

import (
    "fmt"
    "github.com/stretchr/testify/assert"
    "lom/src/lib/lomcommon"
    "lom/src/lib/lomipc"
    "testing"
    "time"
)

func init() {
    configFiles := &lomcommon.ConfigFiles_t{}
    configFiles.GlobalFl = "../../lib/lib_test/config/globals.conf.json"
    configFiles.ActionsFl = "../../lib/lib_test/config/actions.conf.json"
    configFiles.BindingsFl = "../../lib/lib_test/config/actions.conf.json"
    configFiles.ProcsFl = "../../lib/lib_test/config/procs.conf.json"
    lomcommon.InitConfigMgr(configFiles)
}

/* Validate that reportingLimiter reports successfuly for first time for an anomaly key */
func Test_DetectionReportingFreqLimiter_ReportsSuccessfulyForFirstTime(t *testing.T) {
    detectionReportingFrequencyLimiter := GetDefaultDetectionFrequencyLimiter()
    shouldReport := detectionReportingFrequencyLimiter.ShouldReport("Ethernet0")
    cache := detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache
    _, ok := cache["Ethernet0"]

    assert := assert.New(t)
    assert.True(shouldReport, "ShouldReport is expected to be true")
    assert.Equal(1, len(cache), "Length of cache is expected to be 1")
    assert.True(ok)
}

/* Validate that reportingLimiter does not report in initial frequency */
func Test_DetectionReportingFreqLimiter_DoesNotReportForInitialFrequency(t *testing.T) {
    detectionReportingFrequencyLimiter := GetDefaultDetectionFrequencyLimiter()
    currentTimeMinusTwoMins := time.Now().Add(-2 * time.Minute)
    reportingDetails := ReportingDetails{lastReported: currentTimeMinusTwoMins, countOfTimesReported: 8}
    detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"] = &reportingDetails
    shouldReport := detectionReportingFrequencyLimiter.ShouldReport("Ethernet0")

    assert := assert.New(t)
    assert.False(shouldReport, "ShouldReport is expected to be false")
    assert.False(currentTimeMinusTwoMins.Equal(detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"].lastReported), "Cache is expected to have updated.")
    assert.Equal(9, detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"].countOfTimesReported, "CountOfTimesReported is expected to be 9")
}

/* Validate that reportingLimiter reports in initial freq */
func Test_DetectionReportingFreqLimiter_ReportsInInitialFrequency(t *testing.T) {
    detectionReportingFrequencyLimiter := GetDefaultDetectionFrequencyLimiter()
    currentTimeMinusTwoMins := time.Now().Add(-7 * time.Minute)
    reportingDetails := ReportingDetails{lastReported: currentTimeMinusTwoMins, countOfTimesReported: 8}
    detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"] = &reportingDetails
    shouldReport := detectionReportingFrequencyLimiter.ShouldReport("Ethernet0")

    assert := assert.New(t)
    assert.True(shouldReport, "ShouldReport is expected to be True")
    assert.False(currentTimeMinusTwoMins.Equal(detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"].lastReported), "Cache is expected to have updated.")
    assert.Equal(9, detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"].countOfTimesReported, "CountOfTimesReported is expected to be 9")
}

/* Validates that reportingLimiter does not report for subsequent frequency */
func Test_DetectionReportingFreqLimiter_DoesNotReportForSubsequentFrequency(t *testing.T) {
    detectionReportingFrequencyLimiter := GetDefaultDetectionFrequencyLimiter()
    currentTimeMinusTwoMins := time.Now().Add(-2 * time.Minute)
    reportingDetails := ReportingDetails{lastReported: currentTimeMinusTwoMins, countOfTimesReported: 15}
    detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"] = &reportingDetails
    shouldReport := detectionReportingFrequencyLimiter.ShouldReport("Ethernet0")

    assert := assert.New(t)
    assert.False(shouldReport, "ShouldReport is expected to be false")
    assert.False(currentTimeMinusTwoMins.Equal(detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"].lastReported), "Cache is expected to have updated.")
    assert.Equal(16, detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"].countOfTimesReported, "CountOfTimesReported is expected to be 16")
}

/* Validates that reportingLimiter does report in subsequent Frequency */
func Test_LimitDetectionReportingFreq_ReportsInSubsequentFrequency(t *testing.T) {
    detectionReportingFrequencyLimiter := GetDefaultDetectionFrequencyLimiter()
    currentTimeMinusTwoMins := time.Now().Add(-62 * time.Minute)
    reportingDetails := ReportingDetails{lastReported: currentTimeMinusTwoMins, countOfTimesReported: 15}
    detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"] = &reportingDetails
    shouldReport := detectionReportingFrequencyLimiter.ShouldReport("Ethernet0")

    assert := assert.New(t)
    assert.True(shouldReport, "ShouldReport is expected to be True")
    assert.False(currentTimeMinusTwoMins.Equal(detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"].lastReported), "Cache is expected to have updated.")
    assert.Equal(16, detectionReportingFrequencyLimiter.(*PluginReportingFrequencyLimiter).cache["Ethernet0"].countOfTimesReported, "CountOfTimesReported is expected to be 16")
}

type MockElement struct {
    key int
}

/* Validates FixedSizeRollingWindow AddElement does not add more than max allowed elements into the rolling window */
func Test_FixedSizeRollingWindow_AddElementDoesNotAddMoreThanMaxElements(t *testing.T) {
    // Mock
    fixedSizeRollingWindow := FixedSizeRollingWindow{}
    fixedSizeRollingWindow.Initialize(4)

    mockElement1 := MockElement{key: 1}
    mockElement2 := MockElement{key: 2}
    mockElement3 := MockElement{key: 3}
    mockElement4 := MockElement{key: 4}
    mockElement5 := MockElement{key: 5}
    mockElement6 := MockElement{key: 6}

    fixedSizeRollingWindow.AddElement(mockElement1)
    fixedSizeRollingWindow.AddElement(mockElement2)
    fixedSizeRollingWindow.AddElement(mockElement3)
    fixedSizeRollingWindow.AddElement(mockElement4)
    fixedSizeRollingWindow.AddElement(mockElement5)
    fixedSizeRollingWindow.AddElement(mockElement6)

    // Act
    list := fixedSizeRollingWindow.GetElements()

    // Assert.
    validator := 3
    assert := assert.New(t)
    for iterator := list.Front(); iterator != nil; iterator = iterator.Next() {
        mockElmnt := iterator.Value.(MockElement)
        assert.Equal(validator, mockElmnt.key, fmt.Sprintf("Key is expected to be %d", validator))
        validator = validator + 1
    }
    // Ensure the elements are as expected while traversing from back to front.
    validator = 6
    for iterator := list.Back(); iterator != nil; iterator = iterator.Prev() {
        mockElmnt := iterator.Value.(MockElement)
        assert.Equal(validator, mockElmnt.key, fmt.Sprintf("Key is expected to be %d", validator))
        validator = validator - 1
    }
}

/* Validates that FixedSizeRollingWindow Initialize returns error for invalid maxSize */
func Test_FixedSizeRollingWindow_InitializeReturnsErrorForInvalidMaxSize(t *testing.T) {
    // Mock
    fixedSizeRollingWindow := FixedSizeRollingWindow{}
    // Act
    err := fixedSizeRollingWindow.Initialize(0)
    // Assert.
    assert := assert.New(t)
    assert.NotEqual(nil, err, "Error is expected to be non nil for input 0")

    // Act
    err = fixedSizeRollingWindow.Initialize(-1)
    // Assert.
    assert.NotEqual(nil, err, "Error is expected to be non nil for input 1")
}

/* Validates that FixedSizeRollingWindow returns empty list for no addition of elements */
func Test_FixedSizeRollingWindow_InitializeReturnsEmptyListForNoAdditionOfElements(t *testing.T) {
    // Mock
    fixedSizeRollingWindow := FixedSizeRollingWindow{}

    // Act
    err := fixedSizeRollingWindow.Initialize(4)
    list := fixedSizeRollingWindow.GetElements()
    countOfElements := 0
    for iterator := list.Front(); iterator != nil; iterator = iterator.Next() {
        countOfElements = countOfElements + 1
    }

    // Assert.
    assert := assert.New(t)
    assert.Equal(nil, err, "Error is expected to be nil")
    assert.NotEqual(nil, fixedSizeRollingWindow.GetElements(), "DoubleyLinkedList expected to be non nil")
    assert.Equal(0, countOfElements, "CountOfElements expected to be 0")
}

const (
    pluginName = "DummyPlugin"
    version    = "1.0.0.0"
)

var testRequestFrequency int

type DummyPlugin struct {
    testValue1 int
    testValue2 int
    PeriodicDetectionPluginUtil
}

func (dummyPlugin *DummyPlugin) Init(actionConfig *lomcommon.ActionCfg_t) error {
    dummyPlugin.testValue1 = 1
    err := dummyPlugin.PeriodicDetectionPluginUtil.Init(pluginName, testRequestFrequency, actionConfig, dummyPlugin.executeRequest, dummyPlugin.executeShutdown)
    if err != nil {
        return err
    }

    return nil
}

func (dummyPlugin *DummyPlugin) executeRequest(request *lomipc.ActionRequestData, isHealthy *bool) *lomipc.ActionResponseData {
    dummyPlugin.testValue1 = 2
    *isHealthy = true
    if request.Action == "ReturnNilScenario" {
        return nil
    }

    if request.Action == "Sleep" {
        time.Sleep(5 * time.Second)
    }
    return &lomipc.ActionResponseData{}
}

func (dummyPlugin *DummyPlugin) executeShutdown() error {
    dummyPlugin.testValue2 = 3
    return nil
}

func (dummyPlugin *DummyPlugin) GetPluginID() PluginId {
    return PluginId{Name: pluginName, Version: version}
}

/* Validates that Init returns error for invalid arguments */
func Test_PeriodicDetectionPluginUtil_InitReturnsErrorForInvalidArgument(t *testing.T) {
    // Mock
    periodicDetectionPluginUtil := &PeriodicDetectionPluginUtil{}
    testRequestFrequency = 1
    actionConfig := lomcommon.ActionCfg_t{HeartbeatInt: -1}
    err := periodicDetectionPluginUtil.Init("dummyName", 10, &actionConfig, nil, nil)

    // Assert.
    assert := assert.New(t)
    assert.NotNil(err, "err is expected to be non nil")

    actionConfig = lomcommon.ActionCfg_t{HeartbeatInt: 1}
    err = periodicDetectionPluginUtil.Init("dummyName", -10, &actionConfig, nil, nil)

    // Assert.
    assert.NotNil(err, "err is expected to be non nil")

    actionConfig = lomcommon.ActionCfg_t{HeartbeatInt: 1}
    err = periodicDetectionPluginUtil.Init("dummyName", 10, &actionConfig, nil, nil)

    // Assert.
    assert.NotNil(err, "err is expected to be non nil")

    actionConfig = lomcommon.ActionCfg_t{HeartbeatInt: 1}
    dummyPlugin := &DummyPlugin{}
    err = periodicDetectionPluginUtil.Init("dummyName", 10, &actionConfig, nil, dummyPlugin.executeShutdown)

    // Assert.
    assert.NotNil(err, "err is expected to be non nil")
    err = periodicDetectionPluginUtil.Init("dummyName", 10, &actionConfig, dummyPlugin.executeRequest, nil)

    // Assert.
    assert.NotNil(err, "err is expected to be non nil")
}

/* Validates that Request returns failure when actionRequest has timeout */
func Test_PeriodicDetectionPluginUtil_ReturnsErrorWhenTimeoutIsInvalid(t *testing.T) {
    // Mock
    periodicDetectionPluginUtil := &PeriodicDetectionPluginUtil{}
    pluginHBChan := make(chan PluginHeartBeat, 2)
    request := &lomipc.ActionRequestData{Timeout: 10}

    // Act
    response := periodicDetectionPluginUtil.Request(pluginHBChan, request)

    // Assert.
    assert := assert.New(t)
    assert.NotNil(response, "response is expected to be non nil")
    assert.Equal(response.ResultCode, ResultCodeInvalidArgument, "resultCode is expected to be ResultCodeInvalidArgument")
}

/* Validates that Request returns successfully when ActionResponseData is non nil */
func Test_PeriodicDetectionPluginUtil_RequestDetectsSuccessfuly(t *testing.T) {
    // Mock
    var dummyPlugin Plugin
    testRequestFrequency = 1
    dummyPlugin = &DummyPlugin{}
    actionConfig := lomcommon.ActionCfg_t{HeartbeatInt: 3600}
    dummyPlugin.Init(&actionConfig)
    request := &lomipc.ActionRequestData{}
    pluginHBChan := make(chan PluginHeartBeat, 2)

    // Act
    response := dummyPlugin.Request(pluginHBChan, request)

    // Assert.
    assert := assert.New(t)
    assert.NotNil(response, "response is expected to be non nil")
    assert.Equal(2, dummyPlugin.(*DummyPlugin).testValue1, "someValue is expected to be 2")
}

/* Validates that the util sends heartbeat */
func Test_PeriodicDetectionPluginUtil_SendsHeartbeat(t *testing.T) {
    var dummyPlugin Plugin
    testRequestFrequency = 1
    dummyPlugin = &DummyPlugin{}
    actionConfig := lomcommon.ActionCfg_t{HeartbeatInt: 1}
    dummyPlugin.Init(&actionConfig)
    request := &lomipc.ActionRequestData{Action: "ReturnNilScenario"}
    go func() {
        time.Sleep(3 * time.Second)
        dummyPlugin.Shutdown()
    }()
    pluginHBChan := make(chan PluginHeartBeat, 10)
    response := dummyPlugin.Request(pluginHBChan, request)
    <-pluginHBChan
    assert := assert.New(t)
    assert.NotNil(response, "response is expected to be non nil")
    assert.Equal(2, dummyPlugin.(*DummyPlugin).testValue1, "someValue is expected to be 2")
    assert.True(dummyPlugin.(*DummyPlugin).requestAborted, "requestAborted is expected to be true")
}

/* Validates that the request is aborted on shutdown */
func Test_PeriodicDetectionPluginUtil_EnsureRequestAbortedOnShutdown(t *testing.T) {
    var dummyPlugin Plugin
    testRequestFrequency = 2
    dummyPlugin = &DummyPlugin{}
    actionConfig := lomcommon.ActionCfg_t{HeartbeatInt: 3600}
    dummyPlugin.Init(&actionConfig)
    request := &lomipc.ActionRequestData{Action: "ReturnNilScenario"}
    go func() {
        time.Sleep(3 * time.Second)
        dummyPlugin.Shutdown()
    }()
    pluginHBChan := make(chan PluginHeartBeat, 2)
    response := dummyPlugin.Request(pluginHBChan, request)
    assert := assert.New(t)
    assert.NotNil(response, "response is expected to be non nil")
    // Give shutdown 2 seconds to finish its complete execution.
    time.Sleep(2 * time.Second)
    assert.Equal(2, dummyPlugin.(*DummyPlugin).testValue1, "someValue is expected to be 2")
    assert.Equal(3, dummyPlugin.(*DummyPlugin).testValue2, "otherValue is expected to be 3")
    assert.True(dummyPlugin.(*DummyPlugin).requestAborted, "requestAborted is expected to be true")
}

/* Validates that shutdown timesout when request is still active for a long time */
func Test_PeriodicDetectionPluginUtil_EnsureShutDownTimesOut(t *testing.T) {
    var dummyPlugin Plugin
    testRequestFrequency = 1
    dummyPlugin = &DummyPlugin{}
    actionConfig := lomcommon.ActionCfg_t{HeartbeatInt: 3600}
    dummyPlugin.Init(&actionConfig)
    request := &lomipc.ActionRequestData{Action: "Sleep"}
    go func() {
        pluginHBChan := make(chan PluginHeartBeat, 2)
        dummyPlugin.Request(pluginHBChan, request)
    }()
    time.Sleep(2 * time.Second)
    err := dummyPlugin.Shutdown()
    assert := assert.New(t)
    assert.NotNil(err, "err is expected to be non nil")
    assert.Equal(2, dummyPlugin.(*DummyPlugin).testValue1, "someValue is expected to be 2")
    assert.Equal(0, dummyPlugin.(*DummyPlugin).testValue2, "otherValue is expected to be 0")
    assert.False(dummyPlugin.(*DummyPlugin).requestAborted, "requestAborted is expected to be false")
}
