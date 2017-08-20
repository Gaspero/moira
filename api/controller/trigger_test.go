package controller

import (
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/api"
	"github.com/moira-alert/moira-alert/api/dto"
	"github.com/moira-alert/moira-alert/mock/moira-alert"
	"github.com/satori/go.uuid"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"time"
)

func TestSaveTrigger(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	triggerID := uuid.NewV4().String()
	trigger := moira.Trigger{ID: triggerID}
	lastCheck := moira.CheckData{
		Metrics: map[string]moira.MetricState{
			"super.metric1": {},
			"super.metric2": {},
		},
	}
	emptyLastCheck := moira.CheckData{
		Metrics: make(map[string]moira.MetricState, 0),
	}

	Convey("No timeSeries", t, func() {
		Convey("No last check", func() {
			database.EXPECT().AcquireTriggerCheckLock(triggerID, 10)
			database.EXPECT().DeleteTriggerCheckLock(triggerID)
			database.EXPECT().GetTriggerLastCheck(triggerID).Return(nil, nil)
			database.EXPECT().SetTriggerLastCheck(triggerID, gomock.Any()).Return(nil)
			database.EXPECT().SaveTrigger(triggerID, &trigger).Return(nil)
			resp, err := SaveTrigger(database, &trigger, triggerID, make(map[string]bool))
			So(err, ShouldBeNil)
			So(resp, ShouldResemble, &dto.SaveTriggerResponse{ID: triggerID, Message: "trigger updated"})
		})
		Convey("Has last check", func() {
			actualLastCheck := lastCheck
			database.EXPECT().AcquireTriggerCheckLock(triggerID, 10)
			database.EXPECT().DeleteTriggerCheckLock(triggerID)
			database.EXPECT().GetTriggerLastCheck(triggerID).Return(&actualLastCheck, nil)
			database.EXPECT().SetTriggerLastCheck(triggerID, &actualLastCheck).Return(nil)
			database.EXPECT().SaveTrigger(triggerID, &trigger).Return(nil)
			resp, err := SaveTrigger(database, &trigger, triggerID, make(map[string]bool))
			So(err, ShouldBeNil)
			So(resp, ShouldResemble, &dto.SaveTriggerResponse{ID: triggerID, Message: "trigger updated"})
			So(actualLastCheck, ShouldResemble, emptyLastCheck)
		})
	})

	Convey("Has timeSeries", t, func() {
		actualLastCheck := lastCheck
		database.EXPECT().AcquireTriggerCheckLock(triggerID, 10)
		database.EXPECT().DeleteTriggerCheckLock(triggerID)
		database.EXPECT().GetTriggerLastCheck(triggerID).Return(nil, nil)
		database.EXPECT().SetTriggerLastCheck(triggerID, gomock.Any()).Return(nil)
		database.EXPECT().SaveTrigger(triggerID, &trigger).Return(nil)
		resp, err := SaveTrigger(database, &trigger, triggerID, map[string]bool{"super.metric1": true, "super.metric2": true})
		So(err, ShouldBeNil)
		So(resp, ShouldResemble, &dto.SaveTriggerResponse{ID: triggerID, Message: "trigger updated"})
		So(actualLastCheck, ShouldResemble, lastCheck)
	})

	Convey("Errors", t, func() {
		Convey("AcquireTriggerCheckLock error", func() {
			expected := fmt.Errorf("AcquireTriggerCheckLock error")
			database.EXPECT().AcquireTriggerCheckLock(triggerID, 10).Return(expected)
			resp, err := SaveTrigger(database, &trigger, triggerID, make(map[string]bool))
			So(err, ShouldResemble, api.ErrorInternalServer(expected))
			So(resp, ShouldBeNil)
		})

		Convey("GetTriggerLastCheck error", func() {
			expected := fmt.Errorf("GetTriggerLastCheck error")
			database.EXPECT().AcquireTriggerCheckLock(triggerID, 10)
			database.EXPECT().DeleteTriggerCheckLock(triggerID)
			database.EXPECT().GetTriggerLastCheck(triggerID).Return(nil, expected)
			resp, err := SaveTrigger(database, &trigger, triggerID, make(map[string]bool))
			So(err, ShouldResemble, api.ErrorInternalServer(expected))
			So(resp, ShouldBeNil)
		})

		Convey("SetTriggerLastCheck error", func() {
			expected := fmt.Errorf("SetTriggerLastCheck error")
			database.EXPECT().AcquireTriggerCheckLock(triggerID, 10)
			database.EXPECT().DeleteTriggerCheckLock(triggerID)
			database.EXPECT().GetTriggerLastCheck(triggerID).Return(nil, nil)
			database.EXPECT().SetTriggerLastCheck(triggerID, gomock.Any()).Return(expected)
			resp, err := SaveTrigger(database, &trigger, triggerID, make(map[string]bool))
			So(err, ShouldResemble, api.ErrorInternalServer(expected))
			So(resp, ShouldBeNil)
		})

		Convey("SaveTrigger error", func() {
			expected := fmt.Errorf("SaveTrigger error")
			database.EXPECT().AcquireTriggerCheckLock(triggerID, 10)
			database.EXPECT().DeleteTriggerCheckLock(triggerID)
			database.EXPECT().GetTriggerLastCheck(triggerID).Return(nil, nil)
			database.EXPECT().SetTriggerLastCheck(triggerID, gomock.Any()).Return(nil)
			database.EXPECT().SaveTrigger(triggerID, &trigger).Return(expected)
			resp, err := SaveTrigger(database, &trigger, triggerID, make(map[string]bool))
			So(err, ShouldResemble, api.ErrorInternalServer(expected))
			So(resp, ShouldBeNil)
		})
	})
}

func TestGetTrigger(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	triggerID := uuid.NewV4().String()
	trigger := moira.Trigger{ID: triggerID}
	begging := time.Unix(0, 0)
	now := time.Now()
	tomorrow := now.Add(time.Hour * 24)
	yesterday := now.Add(-time.Hour * 24)

	Convey("Has trigger no throttling", t, func() {
		database.EXPECT().GetTrigger(triggerID).Return(&trigger, nil)
		database.EXPECT().GetTriggerThrottlingTimestamps(triggerID).Return(begging, begging)
		actual, err := GetTrigger(database, triggerID)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, &dto.Trigger{Trigger: trigger, Throttling: 0})
	})

	Convey("Has trigger has throttling", t, func() {
		database.EXPECT().GetTrigger(triggerID).Return(&trigger, nil)
		database.EXPECT().GetTriggerThrottlingTimestamps(triggerID).Return(tomorrow, begging)
		actual, err := GetTrigger(database, triggerID)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, &dto.Trigger{Trigger: trigger, Throttling: tomorrow.Unix()})
	})

	Convey("Has trigger has old throttling", t, func() {
		database.EXPECT().GetTrigger(triggerID).Return(&trigger, nil)
		database.EXPECT().GetTriggerThrottlingTimestamps(triggerID).Return(yesterday, begging)
		actual, err := GetTrigger(database, triggerID)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, &dto.Trigger{Trigger: trigger, Throttling: 0})
	})

	Convey("GetTrigger error", t, func() {
		expected := fmt.Errorf("GetTrigger error")
		database.EXPECT().GetTrigger(triggerID).Return(nil, expected)
		actual, err := GetTrigger(database, triggerID)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(actual, ShouldBeNil)
	})

	Convey("No trigger", t, func() {
		database.EXPECT().GetTrigger(triggerID).Return(nil, nil)
		actual, err := GetTrigger(database, triggerID)
		So(err, ShouldResemble, api.ErrorNotFound("Trigger not found"))
		So(actual, ShouldBeNil)
	})
}

func TestDeleteTrigger(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	triggerID := uuid.NewV4().String()

	Convey("Success", t, func() {
		database.EXPECT().DeleteTrigger(triggerID).Return(nil)
		err := DeleteTrigger(database, triggerID)
		So(err, ShouldBeNil)
	})

	Convey("Error", t, func() {
		expected := fmt.Errorf("Oooops! Error delete")
		database.EXPECT().DeleteTrigger(triggerID).Return(expected)
		err := DeleteTrigger(database, triggerID)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
	})
}

func TestGetTriggerThrottling(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	triggerID := uuid.NewV4().String()
	begging := time.Unix(0, 0)
	now := time.Now()
	tomorrow := now.Add(time.Hour * 24)
	yesterday := now.Add(-time.Hour * 24)

	Convey("no throttling", t, func() {
		database.EXPECT().GetTriggerThrottlingTimestamps(triggerID).Return(begging, begging)
		actual, err := GetTriggerThrottling(database, triggerID)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, &dto.ThrottlingResponse{Throttling: 0})
	})

	Convey("has throttling", t, func() {
		database.EXPECT().GetTriggerThrottlingTimestamps(triggerID).Return(tomorrow, begging)
		actual, err := GetTriggerThrottling(database, triggerID)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, &dto.ThrottlingResponse{Throttling: tomorrow.Unix()})
	})

	Convey("has old throttling", t, func() {
		database.EXPECT().GetTriggerThrottlingTimestamps(triggerID).Return(yesterday, begging)
		actual, err := GetTriggerThrottling(database, triggerID)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, &dto.ThrottlingResponse{Throttling: 0})
	})
}

func TestGetTriggerLastCheck(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	triggerID := uuid.NewV4().String()
	lastCheck := moira.CheckData{}

	Convey("Success", t, func() {
		database.EXPECT().GetTriggerLastCheck(triggerID).Return(&lastCheck, nil)
		check, err := GetTriggerLastCheck(database, triggerID)
		So(err, ShouldBeNil)
		So(check, ShouldResemble, &dto.TriggerCheck{
			TriggerID: triggerID,
			CheckData: &lastCheck,
		})
	})

	Convey("Error", t, func() {
		expected := fmt.Errorf("Oooops! Error get")
		database.EXPECT().GetTriggerLastCheck(triggerID).Return(nil, expected)
		check, err := GetTriggerLastCheck(database, triggerID)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(check, ShouldBeNil)
	})
}

func TestDeleteTriggerThrottling(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	triggerID := uuid.NewV4().String()

	Convey("Success", t, func() {
		database.EXPECT().DeleteTriggerThrottling(triggerID).Return(nil)
		err := DeleteTriggerThrottling(database, triggerID)
		So(err, ShouldBeNil)
	})

	Convey("Error", t, func() {
		expected := fmt.Errorf("Oooops! Error delete")
		database.EXPECT().DeleteTriggerThrottling(triggerID).Return(expected)
		err := DeleteTriggerThrottling(database, triggerID)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
	})
}

func TestDeleteTriggerMetric(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	triggerID := uuid.NewV4().String()
	trigger := moira.Trigger{ID: triggerID}
	lastCheck := moira.CheckData{
		Metrics: map[string]moira.MetricState{
			"super.metric1": {},
		},
	}
	emptyLastCheck := moira.CheckData{
		Metrics: make(map[string]moira.MetricState, 0),
	}

	Convey("Success delete from last check", t, func() {
		expectedLastCheck := lastCheck
		database.EXPECT().GetTrigger(triggerID).Return(&trigger, nil)
		database.EXPECT().AcquireTriggerCheckLock(triggerID, 10).Return(nil)
		database.EXPECT().DeleteTriggerCheckLock(triggerID)
		database.EXPECT().GetTriggerLastCheck(triggerID).Return(&expectedLastCheck, nil)
		database.EXPECT().RemovePatternsMetrics(trigger.Patterns).Return(nil)
		database.EXPECT().SetTriggerLastCheck(triggerID, &expectedLastCheck)
		err := DeleteTriggerMetric(database, "super.metric1", triggerID)
		So(err, ShouldBeNil)
		So(expectedLastCheck, ShouldResemble, emptyLastCheck)
	})

	Convey("Success delete nothing to delete", t, func() {
		expectedLastCheck := emptyLastCheck
		database.EXPECT().GetTrigger(triggerID).Return(&trigger, nil)
		database.EXPECT().AcquireTriggerCheckLock(triggerID, 10).Return(nil)
		database.EXPECT().DeleteTriggerCheckLock(triggerID)
		database.EXPECT().GetTriggerLastCheck(triggerID).Return(&expectedLastCheck, nil)
		database.EXPECT().RemovePatternsMetrics(trigger.Patterns).Return(nil)
		database.EXPECT().SetTriggerLastCheck(triggerID, &expectedLastCheck)
		err := DeleteTriggerMetric(database, "super.metric1", triggerID)
		So(err, ShouldBeNil)
		So(expectedLastCheck, ShouldResemble, emptyLastCheck)
	})

	Convey("No trigger", t, func() {
		database.EXPECT().GetTrigger(triggerID).Return(nil, nil)
		err := DeleteTriggerMetric(database, "super.metric1", triggerID)
		So(err, ShouldResemble, api.ErrorInvalidRequest(fmt.Errorf("Trigger not found")))
	})

	Convey("No last check", t, func() {
		database.EXPECT().GetTrigger(triggerID).Return(&trigger, nil)
		database.EXPECT().AcquireTriggerCheckLock(triggerID, 10).Return(nil)
		database.EXPECT().DeleteTriggerCheckLock(triggerID)
		database.EXPECT().GetTriggerLastCheck(triggerID).Return(nil, nil)
		err := DeleteTriggerMetric(database, "super.metric1", triggerID)
		So(err, ShouldResemble, api.ErrorInvalidRequest(fmt.Errorf("Trigger check not found")))
	})

	Convey("Get trigger error", t, func() {
		expected := fmt.Errorf("Get trigger error")
		database.EXPECT().GetTrigger(triggerID).Return(nil, expected)
		err := DeleteTriggerMetric(database, "super.metric1", triggerID)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
	})

	Convey("AcquireTriggerCheckLock error", t, func() {
		expected := fmt.Errorf("Acquire error")
		database.EXPECT().GetTrigger(triggerID).Return(&trigger, nil)
		database.EXPECT().AcquireTriggerCheckLock(triggerID, 10).Return(expected)
		err := DeleteTriggerMetric(database, "super.metric1", triggerID)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
	})

	Convey("GetTriggerLastCheck error", t, func() {
		expected := fmt.Errorf("Last check error")
		database.EXPECT().GetTrigger(triggerID).Return(&trigger, nil)
		database.EXPECT().AcquireTriggerCheckLock(triggerID, 10).Return(nil)
		database.EXPECT().DeleteTriggerCheckLock(triggerID)
		database.EXPECT().GetTriggerLastCheck(triggerID).Return(nil, expected)
		err := DeleteTriggerMetric(database, "super.metric1", triggerID)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
	})

	Convey("RemovePatternsMetrics error", t, func() {
		expected := fmt.Errorf("RemovePatternsMetrics err")
		database.EXPECT().GetTrigger(triggerID).Return(&trigger, nil)
		database.EXPECT().AcquireTriggerCheckLock(triggerID, 10).Return(nil)
		database.EXPECT().DeleteTriggerCheckLock(triggerID)
		database.EXPECT().GetTriggerLastCheck(triggerID).Return(&lastCheck, nil)
		database.EXPECT().RemovePatternsMetrics(trigger.Patterns).Return(expected)
		err := DeleteTriggerMetric(database, "super.metric1", triggerID)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
	})

	Convey("SetTriggerLastCheck error", t, func() {
		expected := fmt.Errorf("RemovePatternsMetrics err")
		database.EXPECT().GetTrigger(triggerID).Return(&trigger, nil)
		database.EXPECT().AcquireTriggerCheckLock(triggerID, 10).Return(nil)
		database.EXPECT().DeleteTriggerCheckLock(triggerID)
		database.EXPECT().GetTriggerLastCheck(triggerID).Return(&lastCheck, nil)
		database.EXPECT().RemovePatternsMetrics(trigger.Patterns).Return(nil)
		database.EXPECT().SetTriggerLastCheck(triggerID, &lastCheck).Return(expected)
		err := DeleteTriggerMetric(database, "super.metric1", triggerID)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
	})
}

func TestSetMetricsMaintenance(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	database := mock_moira_alert.NewMockDatabase(mockCtrl)
	triggerID := uuid.NewV4().String()
	maintenance := make(map[string]int64)

	Convey("Success", t, func() {
		database.EXPECT().SetTriggerMetricsMaintenance(triggerID, maintenance).Return(nil)
		err := SetMetricsMaintenance(database, triggerID, maintenance)
		So(err, ShouldBeNil)
	})

	Convey("Error", t, func() {
		expected := fmt.Errorf("Oooops! Error set")
		database.EXPECT().SetTriggerMetricsMaintenance(triggerID, maintenance).Return(expected)
		err := SetMetricsMaintenance(database, triggerID, maintenance)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
	})
}