package handlers

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strconv"
	"testing"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/broker"
	"github.com/DedovInside/AutoInspect/backend/internal/config"
	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/DedovInside/AutoInspect/backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const (
	handlerAnalysisMake       = "Volkswagen"
	handlerAnalysisModel      = "Golf"
	handlerAnalysisGeneration = "5"
	handlerAnalysisYear       = 2015
	handlerAnalysisTopic      = "autoinspect.analysis.request"
	handlerUploadBucket       = "autoinspect-uploads"
	handlerModelBucket        = "autoinspect-models"
)

func TestAnalysisHandlerSubmit(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	userID := uuid.New()
	jobRepo := &handlerAnalysisJobRepo{}
	fileRepo := &handlerAnalysisFileRepo{}
	publisher := &handlerAnalysisPublisher{}
	handler := NewAnalysisHandler(
		service.NewAnalysisService(
			fileRepo,
			jobRepo,
			handlerUniversalModelRepo(),
			nil,
			publisher,
			nil,
			&config.S3Config{BucketUploads: handlerUploadBucket, BucketModels: handlerModelBucket},
			&config.KafkaConfig{TopicAnalysisRequest: handlerAnalysisTopic},
		),
		nil,
		nil,
		3,
		5,
		[]string{"image/jpeg"},
		nil,
	)

	router := gin.New()
	router.POST("/v1/analyses", withTestUser(userID, domain.RoleUser), handler.Submit)

	body, contentType := multipartAnalysisBody(t)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/analyses", body)
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("Idempotency-Key", "analysis-handler-key")

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusAccepted, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"status":"pending"`)
	require.Contains(t, recorder.Body.String(), `"car_make":"Volkswagen"`)
	require.NotNil(t, jobRepo.created)
	require.Equal(t, userID, jobRepo.created.UserID)
	require.Len(t, fileRepo.uploaded, 1)
	require.Len(t, publisher.messages, 1)
}

func TestAnalysisHandlerGetByID(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	userID := uuid.New()
	job := handlerAnalysisJob(userID, uuid.New(), domain.StatusCompleted)
	handler := NewAnalysisHandler(
		service.NewAnalysisService(
			&handlerAnalysisFileRepo{},
			&handlerAnalysisJobRepo{byID: map[uuid.UUID]*domain.AnalysisJob{job.ID: job}},
			handlerUniversalModelRepo(),
			nil,
			nil,
			nil,
			&config.S3Config{},
			&config.KafkaConfig{},
		),
		nil,
		nil,
		3,
		5,
		[]string{"image/jpeg"},
		nil,
	)

	router := gin.New()
	router.GET("/v1/analyses/:id", withTestUser(userID, domain.RoleUser), handler.GetByID)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/analyses/"+job.ID.String(), http.NoBody)

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"status":"completed"`)
	require.Contains(t, recorder.Body.String(), `"car_model":"Golf"`)
}

func TestAnalysisHandlerListMine(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	userID := uuid.New()
	job := handlerAnalysisJob(userID, uuid.New(), domain.StatusPending)
	handler := NewAnalysisHandler(
		service.NewAnalysisService(nil, &handlerAnalysisJobRepo{
			byID: map[uuid.UUID]*domain.AnalysisJob{job.ID: job},
		}, nil, nil, nil, nil, nil, nil),
		nil,
		nil,
		3,
		5,
		[]string{"image/jpeg"},
		nil,
	)

	router := gin.New()
	router.GET("/v1/analyses", withTestUser(userID, domain.RoleUser), handler.ListMine)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/analyses?limit=20&offset=0", http.NoBody)

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"items"`)
	require.Contains(t, recorder.Body.String(), `"car_make":"Volkswagen"`)
}

func TestAnalysisHandlerGetPresignedImageURL(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	userID := uuid.New()
	job := handlerAnalysisJob(userID, uuid.New(), domain.StatusCompleted)
	job.ImageKeys = []string{"uploads/user/car.jpg"}
	handler := NewAnalysisHandler(
		service.NewAnalysisService(
			&handlerAnalysisFileRepo{presignedURL: "https://cdn.example.test/car.jpg"},
			&handlerAnalysisJobRepo{byID: map[uuid.UUID]*domain.AnalysisJob{job.ID: job}},
			nil,
			nil,
			nil,
			nil,
			&config.S3Config{BucketUploads: handlerUploadBucket, PresignedURLTTL: time.Minute},
			nil,
		),
		nil,
		nil,
		3,
		5,
		[]string{"image/jpeg"},
		nil,
	)

	router := gin.New()
	router.GET("/v1/analyses/:id/images/:idx/url", withTestUser(userID, domain.RoleUser), handler.GetPresignedImageURL)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/analyses/"+job.ID.String()+"/images/0/url", http.NoBody)

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"url":"https://cdn.example.test/car.jpg"`)
}

func TestAnalysisHandlerFindMatchingCarServices(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	userID := uuid.New()
	job := handlerAnalysisJob(userID, uuid.New(), domain.StatusCompleted)
	profileID := uuid.New()
	matchSvc := service.NewCarServiceMatchingService(
		&handlerAnalysisJobRepo{byID: map[uuid.UUID]*domain.AnalysisJob{job.ID: job}},
		&handlerMatchRepo{matches: []*domain.CarServiceMatch{
			{
				Profile: &domain.CarServiceProfile{
					ID:               profileID,
					OrganizationName: "Detail Lab",
					City:             "Москва",
					Address:          "Тестовая улица, 1",
					IsActive:         true,
				},
				MatchCount: 1,
			},
		}},
		nil,
		nil,
		nil,
	)
	handler := NewAnalysisHandler(nil, matchSvc, nil, 3, 5, []string{"image/jpeg"}, nil)

	router := gin.New()
	router.GET("/v1/analyses/:id/matching-services", withTestUser(userID, domain.RoleUser), handler.FindMatchingCarServices)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/v1/analyses/"+job.ID.String()+"/matching-services?limit=20&offset=0",
		http.NoBody,
	)

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"organization_name":"Detail Lab"`)
	require.Contains(t, recorder.Body.String(), `"required_count":1`)
}

func handlerUniversalModelRepo() *handlerCarModelRepo {
	repo := newHandlerCarModelRepo()
	repo.models = append(repo.models, &domain.CarModel{
		ID:                uuid.New(),
		Make:              "general",
		Model:             "general",
		Generation:        "general",
		YearFrom:          1900,
		YearTo:            2100,
		PartsModelS3Key:   "models/general/parts_segmentation.pt",
		PartsConfigS3Key:  "models/general/parts_inference_config.json",
		PartsCatalogS3Key: "models/general/parts_catalog.json",
		ModelVersion:      "v1",
		IsUniversal:       true,
		IsActive:          true,
		CreatedAt:         time.Now().UTC(),
	})
	return repo
}

func handlerAnalysisJob(userID, jobID uuid.UUID, status domain.JobStatus) *domain.AnalysisJob {
	return &domain.AnalysisJob{
		ID:            jobID,
		UserID:        userID,
		CarMake:       handlerAnalysisMake,
		CarModel:      handlerAnalysisModel,
		CarGeneration: handlerAnalysisGeneration,
		CarYear:       handlerAnalysisYear,
		ImageKeys:     []string{"uploads/user/car.jpg"},
		CorrelationID: uuid.New(),
		Status:        status,
		RequestedAt:   time.Now().UTC(),
		Result: &domain.AnalysisResult{
			Results: []domain.ImageAnalysisResult{
				{
					PartsSummary: []domain.PartSummary{
						{
							Name:        "hood",
							ParentName:  "hood",
							DamageCount: 1,
							DamageTypes: []domain.DamageTypeSummary{
								{Code: "dent", NameRU: "вмятина", Count: 1},
							},
						},
					},
				},
			},
		},
	}
}

func multipartAnalysisBody(t *testing.T) (body *bytes.Buffer, contentType string) {
	t.Helper()

	body = &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	require.NoError(t, writer.WriteField("make", handlerAnalysisMake))
	require.NoError(t, writer.WriteField("model", handlerAnalysisModel))
	require.NoError(t, writer.WriteField("generation", handlerAnalysisGeneration))
	require.NoError(t, writer.WriteField("year", strconv.Itoa(handlerAnalysisYear)))

	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="images"; filename="car.jpg"`)
	header.Set("Content-Type", "image/jpeg")
	part, err := writer.CreatePart(header)
	require.NoError(t, err)
	_, err = part.Write([]byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00\x01\x01"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	return body, writer.FormDataContentType()
}

type handlerAnalysisFileRepo struct {
	uploaded     []string
	presignedURL string
}

func (r *handlerAnalysisFileRepo) Upload(_ context.Context, _, objectKey string, data io.Reader, _ string, _ int64) error {
	_, _ = io.ReadAll(data)
	r.uploaded = append(r.uploaded, objectKey)
	return nil
}

func (r *handlerAnalysisFileRepo) Download(context.Context, string, string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(nil)), nil
}

func (r *handlerAnalysisFileRepo) Exists(context.Context, string, string) (bool, error) {
	return true, nil
}

func (r *handlerAnalysisFileRepo) Delete(context.Context, string, string) error {
	return nil
}

func (r *handlerAnalysisFileRepo) GetPresignedURL(context.Context, string, string, time.Duration) (string, error) {
	if r.presignedURL != "" {
		return r.presignedURL, nil
	}
	return "https://cdn.example.test/object.jpg", nil
}

type handlerAnalysisPublisher struct {
	messages []broker.Message
}

func (p *handlerAnalysisPublisher) Publish(_ context.Context, message broker.Message) error {
	p.messages = append(p.messages, message)
	return nil
}

func (p *handlerAnalysisPublisher) Close() error {
	return nil
}

type handlerMatchRepo struct {
	matches []*domain.CarServiceMatch
}

func (r *handlerMatchRepo) FindMatching(
	context.Context,
	[]domain.CarServiceMatchCriterion,
	int,
	int,
) ([]*domain.CarServiceMatch, error) {
	return r.matches, nil
}
