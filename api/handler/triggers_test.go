package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	metricSource "github.com/moira-alert/moira/metric_source"
	mock_metric_source "github.com/moira-alert/moira/mock/metric_source"

	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/api/middleware"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/xiam/to"
)

func TestGetSearchRequestString(t *testing.T) {
	Convey("Given a search request string", t, func() {
		Convey("The value should be converted into lower case", func() {
			testCases := []struct {
				text                  string
				expectedSearchRequest string
			}{
				{"query", "query"},
				{"QUERY", "query"},
				{"Query", "query"},
				{"QueRy", "query"},
			}
			for _, testCase := range testCases {
				req, _ := http.NewRequest("GET", fmt.Sprintf("/api/trigger/search?onlyProblems=false&p=0&size=20&text=%s", testCase.text), nil)
				searchRequest := getSearchRequestString(req)
				So(searchRequest, ShouldEqual, testCase.expectedSearchRequest)
			}
		})
	})
}

func TestGetTriggerFromRequest(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	localSource := mock_metric_source.NewMockMetricSource(mockCtrl)
	remoteSource := mock_metric_source.NewMockMetricSource(mockCtrl)
	fetchResult := mock_metric_source.NewMockFetchResult(mockCtrl)
	sourceProvider := metricSource.CreateMetricSourceProvider(localSource, remoteSource)

	localSource.EXPECT().IsConfigured().Return(true, nil).AnyTimes()
	localSource.EXPECT().GetMetricsTTLSeconds().Return(int64(3600)).AnyTimes()
	localSource.EXPECT().Fetch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(fetchResult, nil).AnyTimes()
	fetchResult.EXPECT().GetPatterns().Return(make([]string, 0), nil).AnyTimes()
	fetchResult.EXPECT().GetMetricsData().Return([]metricSource.MetricData{*metricSource.MakeMetricData("", []float64{}, 0, 0)}).AnyTimes()

	Convey("Given a correct payload", t, func() {
		triggerWarnValue := 0.0
		triggerErrorValue := 1.0
		ttlState := moira.TTLState("NODATA")
		triggerDTO := dto.Trigger{
			TriggerModel: dto.TriggerModel{
				ID:             "test_id",
				Name:           "Test trigger",
				Desc:           new(string),
				Targets:        []string{"foo.bar"},
				WarnValue:      &triggerWarnValue,
				ErrorValue:     &triggerErrorValue,
				TriggerType:    "rising",
				Tags:           []string{"Normal", "DevOps", "DevOpsGraphite-duty"},
				TTLState:       &ttlState,
				TTL:            0,
				Schedule:       &moira.ScheduleData{},
				Expression:     "",
				Patterns:       []string{},
				IsRemote:       false,
				MuteNewMetrics: false,
				AloneMetrics:   map[string]bool{},
				CreatedAt:      &time.Time{},
				UpdatedAt:      &time.Time{},
			},
		}
		body, _ := json.Marshal(triggerDTO)

		request := httptest.NewRequest(http.MethodPut, "/trigger", bytes.NewReader(body))
		request.Header.Add("content-type", "application/json")
		request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "metricSourceProvider", sourceProvider))

		Convey("It should be parsed successfully", func() {
			trigger, err := getTriggerFromRequest(request)
			So(err, ShouldBeNil)
			So(trigger, ShouldResemble, &triggerDTO)
		})
	})

	Convey("Given an incorrect payload", t, func() {
		body := `{
			"name": "test",
			"desc": "",
			"targets": ["foo.bar"],
			"tags": ["test"],
			"patterns": [],
			"expression": "",
			"ttl": 600,
			"ttl_state": "NODATA",
			"sched": {
				"startOffset": 0,
				"endOffset": 1439,
				"tzOffset": -240,
				"days": null
			},
			"is_remote": false,
			"error_value": 1,
			"warn_value": 0,
			"trigger_type": "rising",
			"mute_new_metrics": false,
			"alone_metrics": "beliberda"
		}`

		request := httptest.NewRequest(http.MethodPut, "/trigger", strings.NewReader(body))
		request.Header.Add("content-type", "application/json")
		request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "metricSourceProvider", sourceProvider))

		Convey("Parser should return en error", func() {
			_, err := getTriggerFromRequest(request)
			So(err, ShouldHaveSameTypeAs, api.ErrorInvalidRequest(fmt.Errorf("")))
		})
	})
}

func TestGetMetricTTLByTrigger(t *testing.T) {
	request := httptest.NewRequest(http.MethodPut, "/trigger/new", strings.NewReader(""))
	request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "localMetricTTL", to.Duration("65m")))
	request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "remoteMetricTTL", to.Duration("168h")))

	Convey("Given a local trigger", t, func() {
		trigger := dto.Trigger{TriggerModel: dto.TriggerModel{
			IsRemote: false,
		}}

		Convey("It's metric ttl should be equal to local", func() {
			So(getMetricTTLByTrigger(request, &trigger), ShouldEqual, 65*time.Minute)
		})
	})

	Convey("Given a remote trigger", t, func() {
		trigger := dto.Trigger{TriggerModel: dto.TriggerModel{
			IsRemote: true,
		}}

		Convey("It's metric ttl should be equal to remote", func() {
			So(getMetricTTLByTrigger(request, &trigger), ShouldEqual, 168*time.Hour)
		})
	})
}

func TestTriggerCheckHandler(t *testing.T) {
	Convey("Test triggerCheck handler", t, func() {
		Convey("Checking target metric ttl validation", func() {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			responseWriter := httptest.NewRecorder()

			localSource := mock_metric_source.NewMockMetricSource(mockCtrl)
			remoteSource := mock_metric_source.NewMockMetricSource(mockCtrl)
			fetchResult := mock_metric_source.NewMockFetchResult(mockCtrl)
			sourceProvider := metricSource.CreateMetricSourceProvider(localSource, remoteSource)

			localSource.EXPECT().IsConfigured().Return(true, nil).AnyTimes()
			localSource.EXPECT().GetMetricsTTLSeconds().Return(int64(3600)).AnyTimes()
			localSource.EXPECT().Fetch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(fetchResult, nil).AnyTimes()
			fetchResult.EXPECT().GetPatterns().Return(make([]string, 0), nil).AnyTimes()
			fetchResult.EXPECT().GetMetricsData().Return([]metricSource.MetricData{*metricSource.MakeMetricData("", []float64{}, 0, 0)}).AnyTimes()

			remoteSource.EXPECT().IsConfigured().Return(true, nil).AnyTimes()
			remoteSource.EXPECT().GetMetricsTTLSeconds().Return(int64(604800)).AnyTimes()
			remoteSource.EXPECT().Fetch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(fetchResult, nil).AnyTimes()

			testCases := []struct {
				isRemote         bool
				targets          []string
				expectedResponse string
			}{
				{
					false,
					[]string{
						"integralByInterval(aliasSub(sum(aliasByNode(my.own.metric, 6)), '(.*)', 'metric'), '1h')",
					},
					"{\"targets\":[{\"syntax_ok\":true}]}\n",
				},
				{
					false,
					[]string{
						"integralByInterval(aliasSub(sum(aliasByNode(my.own.metric, 6)), '(.*)', 'metric'), '6h')",
					},
					"{\"targets\":[{\"syntax_ok\":true,\"tree_of_problems\":{\"argument\":\"integralByInterval\",\"position\":0,\"problems\":[{\"argument\":\"6h\",\"type\":\"bad\",\"description\":\"The function integralByInterval has a time sampling parameter 6h larger than allowed by the config:1h5m0s\",\"position\":1}]}}]}\n",
				},
				{
					true,
					[]string{
						"integralByInterval(aliasSub(sum(aliasByNode(my.own.metric, 6)), '(.*)', 'metric'), '1h')",
					},
					"{\"targets\":[{\"syntax_ok\":true}]}\n",
				},
				{
					true,
					[]string{
						"integralByInterval(aliasSub(sum(aliasByNode(my.own.metric, 6)), '(.*)', 'metric'), '6h')",
					},
					"{\"targets\":[{\"syntax_ok\":true}]}\n",
				},
			}
			for n, testCase := range testCases {
				Convey(fmt.Sprintf("TestCase #%d", n), func() {
					triggerWarnValue := float64(10)
					triggerErrorValue := float64(15)
					triggerDTO := dto.Trigger{
						TriggerModel: dto.TriggerModel{
							Name:       "Test trigger",
							Tags:       []string{"Normal", "DevOps", "DevOpsGraphite-duty"},
							WarnValue:  &triggerWarnValue,
							ErrorValue: &triggerErrorValue,
							Targets:    testCase.targets,
							IsRemote:   testCase.isRemote,
						},
					}
					jsonTrigger, _ := json.Marshal(triggerDTO)
					testRequest := httptest.NewRequest(http.MethodPut, "/trigger/new", bytes.NewBuffer(jsonTrigger))
					testRequest.Header.Add("content-type", "application/json")
					testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "metricSourceProvider", sourceProvider))
					testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "localMetricTTL", to.Duration("65m")))
					testRequest = testRequest.WithContext(middleware.SetContextValueForTest(testRequest.Context(), "remoteMetricTTL", to.Duration("168h")))

					triggerCheck(responseWriter, testRequest)

					response := responseWriter.Result()
					defer response.Body.Close()

					contentBytes, _ := io.ReadAll(response.Body)
					contents := string(contentBytes)

					So(contents, ShouldEqual, testCase.expectedResponse)
				})
			}
		})
	})
}
