package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/IdeaEvolver/cutter-pkg/clog"
)

type StatusRequest struct {
	Service string `json:"service"`
}

type StatusLog struct {
	Service string `json:"service"`
	Status  string `json:"status"`
}

func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	service := r.URL.Query().Get("service")

	return h.Statuses.GetStatus(r.Context(), service)
}

func (h *Handler) GetAllStatuses(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	return h.Statuses.GetAllStatuses(r.Context())
}

func (h *Handler) AllChecks(ctx context.Context, bucket, object string) error {
	statuses := []StatusLog{}
	for {

		platformStatus, err := h.Healthchecks.PlatformStatus(ctx)
		if err != nil {
			clog.Fatalf("Error retrieving platform status", err)
		}

		statuses = append(statuses, StatusLog{Service: "platform-api", Status: platformStatus.Status})

		if err := h.Statuses.UpdateStatus(ctx, "platform", platformStatus.Status); err != nil {
			clog.Fatalf("Error updating platform status", err)
		}

		fulfillmentStatus, err := h.Healthchecks.FulfillmentStatus(ctx)
		if err != nil {
			clog.Fatalf("Error retrieving fulfillment status", err)
		}

		statuses = append(statuses, StatusLog{Service: "fulfillment-api", Status: fulfillmentStatus.Status})

		if err := h.Statuses.UpdateStatus(ctx, "fulfillment", fulfillmentStatus.Status); err != nil {
			clog.Fatalf("Error updating fulfillment status", err)
		}

		crmStatus, err := h.Healthchecks.CrmStatus(ctx)
		if err != nil {
			clog.Fatalf("Error retrieving crm status", err)
		}

		statuses = append(statuses, StatusLog{Service: "crm-api", Status: crmStatus.Status})

		if err := h.Statuses.UpdateStatus(ctx, "crm", crmStatus.Status); err != nil {
			clog.Fatalf("Error updating crm status", err)
		}

		studyStatus, err := h.Healthchecks.StudyStatus(ctx)
		if err != nil {
			clog.Fatalf("Error retrieving study status", err)
		}

		statuses = append(statuses, StatusLog{Service: "study-service-api", Status: studyStatus.Status})

		if err := h.Statuses.UpdateStatus(ctx, "study", studyStatus.Status); err != nil {
			clog.Fatalf("Error updating study status", err)
		}

		nodeMetrics, err := h.Metrics.GetNodeMetrics(ctx)
		if err != nil {
			clog.Fatalf("Error retrieving node metrics", err)
		}

		infra := "Ok"
		if !nodeMetrics.Healthy() {
			infra = "high utilization"
		}

		statuses = append(statuses, StatusLog{Service: "infra", Status: infra})
		if err := h.Statuses.UpdateStatus(ctx, "infrastructure", infra); err != nil {
			clog.Fatalf("Error updating infra status", err)
		}

		for _, status := range statuses {
			if err := h.write(ctx, status, bucket, object); err != nil {
				clog.Errorf("unable to write data to bucket %s, object %s:  %v", bucket, object, err)
				return err
			}
		}

		time.Sleep(60 * time.Second)
	}

}

func (h *Handler) write(ctx context.Context, status StatusLog, bucket, object string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()

	wc := h.Storage.Bucket(bucket).Object(object).NewWriter(ctx)
	wc.ContentType = "text/plain"
	wc.Metadata = map[string]string{
		"x-goog-meta-foo": "foo",
		"x-goog-meta-bar": "bar",
	}

	d, err := json.Marshal(status)
	if err != nil {
		return err
	}

	if _, err := wc.Write([]byte(d)); err != nil {
		clog.Errorf("unable to write data to bucket %s, object %s:  %v", bucket, object, err)
		return err
	}

	if err := wc.Close(); err != nil {
		clog.Errorf("unable to close bucket %s, object %s : %v", bucket, object, err)
		return err
	}

	return nil
}
