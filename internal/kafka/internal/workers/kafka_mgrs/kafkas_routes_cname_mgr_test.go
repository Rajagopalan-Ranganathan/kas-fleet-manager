package kafka_mgrs

import (
	"testing"

	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

func TestKafkaRoutesCNAMEManager(t *testing.T) {
	testChangeID := "1234"
	testChangeINSYNC := route53.ChangeStatusInsync

	type fields struct {
		kafkaService services.KafkaService
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "should success when there are no errors",
			fields: fields{kafkaService: &services.KafkaServiceMock{
				ListKafkasWithRoutesNotCreatedFunc: func() ([]*dbapi.KafkaRequest, *errors.ServiceError) {
					kafka := &dbapi.KafkaRequest{
						Name:          "test",
						RoutesCreated: false,
					}
					return []*dbapi.KafkaRequest{
						kafka,
					}, nil
				},
				ChangeKafkaCNAMErecordsFunc: func(kafkaRequest *dbapi.KafkaRequest, action services.KafkaRoutesAction) (*route53.ChangeResourceRecordSetsOutput, *errors.ServiceError) {
					return &route53.ChangeResourceRecordSetsOutput{
						ChangeInfo: &route53.ChangeInfo{
							Id:     &testChangeID,
							Status: &testChangeINSYNC,
						},
					}, nil
				},
				UpdateFunc: func(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
					if !kafkaRequest.RoutesCreated {
						return errors.GeneralError("RoutesCreated is set to true")
					}
					return nil
				},
			}},
			wantErr: false,
		},
		{
			name: "should return error when list kafkas failed",
			fields: fields{kafkaService: &services.KafkaServiceMock{
				ListKafkasWithRoutesNotCreatedFunc: func() ([]*dbapi.KafkaRequest, *errors.ServiceError) {
					return nil, errors.GeneralError("failed to list kafkas")
				},
			}},
			wantErr: true,
		},
		{
			name: "should return error when creating CNAME failed",
			fields: fields{kafkaService: &services.KafkaServiceMock{
				ListKafkasWithRoutesNotCreatedFunc: func() ([]*dbapi.KafkaRequest, *errors.ServiceError) {
					kafka := &dbapi.KafkaRequest{
						Name:          "test",
						RoutesCreated: false,
					}
					return []*dbapi.KafkaRequest{
						kafka,
					}, nil
				},
				ChangeKafkaCNAMErecordsFunc: func(kafkaRequest *dbapi.KafkaRequest, action services.KafkaRoutesAction) (*route53.ChangeResourceRecordSetsOutput, *errors.ServiceError) {
					return nil, errors.GeneralError("failed to create CNAME")
				},
			}},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			k := &KafkaRoutesCNAMEManager{
				kafkaService: test.fields.kafkaService,
				kafkaConfig:  &config.KafkaConfig{EnableKafkaExternalCertificate: true},
			}

			errs := k.Reconcile()
			if len(errs) > 0 && !test.wantErr {
				t.Errorf("unexpected error when reconcile kafka routes: %v", errs)
			}
		})
	}
}
