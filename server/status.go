package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/IdeaEvolver/cutter-pkg/clog"
	"github.com/gocarina/gocsv"
)

type StatusRequest struct {
	Service string `json:"service"`
}

type StatusLog struct {
	Service   string    `json:"service"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	service := r.URL.Query().Get("service")

	return h.Statuses.GetStatus(r.Context(), service)
}

func (h *Handler) GetAllStatuses(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	return h.Statuses.GetAllStatuses(r.Context())
}

func (h *Handler) AllChecks(ctx context.Context, bucket string) error {
	statuses := []*StatusLog{}
	for {

		// platformStatus, err := h.Healthchecks.PlatformStatus(ctx)
		// if err != nil {
		// 	clog.Fatalf("Error retrieving platform status", err)
		// }

		// statuses = append(statuses, &StatusLog{Service: "platform-api", Status: platformStatus.Status})

		// if err := h.Statuses.UpdateStatus(ctx, "platform", platformStatus.Status); err != nil {
		// 	clog.Fatalf("Error updating platform status", err)
		// }

		// fulfillmentStatus, err := h.Healthchecks.FulfillmentStatus(ctx)
		// if err != nil {
		// 	clog.Fatalf("Error retrieving fulfillment status", err)
		// }

		// statuses = append(statuses, &StatusLog{Service: "fulfillment-api", Status: fulfillmentStatus.Status})

		// if err := h.Statuses.UpdateStatus(ctx, "fulfillment", fulfillmentStatus.Status); err != nil {
		// 	clog.Fatalf("Error updating fulfillment status", err)
		// }

		// crmStatus, err := h.Healthchecks.CrmStatus(ctx)
		// if err != nil {
		// 	clog.Fatalf("Error retrieving crm status", err)
		// }

		// statuses = append(statuses, &StatusLog{Service: "crm-api", Status: crmStatus.Status})

		// if err := h.Statuses.UpdateStatus(ctx, "crm", crmStatus.Status); err != nil {
		// 	clog.Fatalf("Error updating crm status", err)
		// }

		// studyStatus, err := h.Healthchecks.StudyStatus(ctx)
		// if err != nil {
		// 	clog.Fatalf("Error retrieving study status", err)
		// }

		// statuses = append(statuses, &StatusLog{Service: "study-service-api", Status: studyStatus.Status})

		// if err := h.Statuses.UpdateStatus(ctx, "study", studyStatus.Status); err != nil {
		// 	clog.Fatalf("Error updating study status", err)
		// }

		nodeMetrics, err := h.Metrics.GetNodeMetrics(ctx)
		if err != nil {
			clog.Fatalf("Error retrieving node metrics", err)
		}

		infra := "Ok"
		if !nodeMetrics.Healthy() {
			infra = "high utilization"
		}

		statuses = append(statuses, &StatusLog{Service: "infra", Status: infra})

		if err := h.Statuses.UpdateStatus(ctx, "infrastructure", infra); err != nil {
			clog.Fatalf("Error updating infra status", err)
		}

		hibbertStatus, err := h.Healthchecks.HibbertStatus(ctx)
		if err != nil {
			clog.Fatalf("Error retrieving hibbert status", err)
		}

		statuses = append(statuses, &StatusLog{Service: "hibbert-api", Status: hibbertStatus.Status})

		if err := h.Statuses.UpdateStatus(ctx, "hibbert", hibbertStatus.Status); err != nil {
			clog.Fatalf("Error updating hibbert status", err)
		}

		azCrmStatus, err := h.Healthchecks.AZCRMStatus(ctx)
		if err != nil {
			clog.Fatalf("Error retrieving az crm status", err)
		}

		statuses = append(statuses, &StatusLog{Service: "azcrm-api", Status: azCrmStatus.Status})

		if err := h.Statuses.UpdateStatus(ctx, "az_crm", azCrmStatus.Status); err != nil {
			clog.Fatalf("Error updating az crm status", err)
		}
		//TODO Ui statuses
		statuses = append(statuses, &StatusLog{Service: "study-ui", Status: "200"})
		statuses = append(statuses, &StatusLog{Service: "platform-ui", Status: "200"})

		for _, status := range statuses {
			if !strings.Contains(status.Status, "200") && !strings.Contains(status.Status, "201") && !strings.Contains(status.Status, "Ok") {
				filename := status.Service + "-logs.csv"
				status.Timestamp = time.Now().UTC()

				clog.Infow("status %s service %s ", status.Status, status.Service)
				if err := h.Statuses.UpdateServiceDown(ctx, status.Service, status.Status, status.Timestamp); err != nil {
					clog.Errorw("unable to insert new down status %v", err)
					return err
				}

				statusReports, err := h.Statuses.GetServiceDown(ctx, status.Service)
				if err != nil {
					clog.Errorw("unable to return service report from table %v", err)
					return err
				}
				fmt.Println("status Reports ", &statusReports[0])
				csvContent, err := gocsv.MarshalString(&statusReports)
				if err != nil {
					clog.Errorw("unable to marshal csv string %v", err)
					return err
				}

				if err := h.Write(ctx, csvContent, bucket, filename); err != nil {
					clog.Errorf("unable to write data to bucket %s, object %s:  %v", bucket, filename, err)
					return err
				}
			}
		}

		time.Sleep(60 * time.Second)
	}

}

func (h *Handler) Write(ctx context.Context, status string, bucket, object string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()

	wc := h.Storage.Bucket(bucket).Object(object).NewWriter(ctx)
	wc.ContentType = "text/csv"
	wc.Metadata = map[string]string{
		"x-goog-meta-foo": "foo",
		"x-goog-meta-bar": "bar",
	}

	if _, err := wc.Write([]byte(status)); err != nil {
		clog.Errorf("unable to write data to bucket %s, object %s:  %v", bucket, object, err)
		return err
	}

	if err := wc.Close(); err != nil {
		clog.Errorf("unable to close bucket %s, object %s : %v", bucket, object, err)
		return err
	}

	return nil
}
