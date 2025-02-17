package observatorium

import (
	"context"
	"encoding/json"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/observatorium"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/flags"

	"github.com/golang-jwt/jwt/v4"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

func NewRunMetricsQueryRangeCommand(env *environments.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query_range",
		Short: "Get metrics with timeseries query range by kafka id from Observatorium",
		Run: func(cmd *cobra.Command, args []string) {
			runGetMetricsByRangeQuery(env, cmd, args)
		},
	}
	cmd.Flags().String(FlagID, "", "Kafka id")
	cmd.Flags().String(FlagOwner, "", "Username")

	return cmd
}
func runGetMetricsByRangeQuery(env *environments.Env, cmd *cobra.Command, _args []string) {
	id := flags.MustGetDefinedString(FlagID, cmd.Flags())
	owner := flags.MustGetDefinedString(FlagOwner, cmd.Flags())
	var observatoriumService services.ObservatoriumService
	env.MustResolveAll(&observatoriumService)

	kafkaMetrics := &observatorium.KafkaMetrics{}
	// create jwt with claims and set it in the context
	jwt := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"username": owner,
	})
	ctx := auth.SetTokenInContext(context.TODO(), jwt)
	params := observatorium.MetricsReqParams{}
	params.ResultType = observatorium.RangeQuery
	params.FillDefaults()

	kafkaId, err := observatoriumService.GetMetricsByKafkaId(ctx, kafkaMetrics, id, params)
	if err != nil {
		glog.Error("An error occurred while attempting to get metrics data ", err.Error())
		return
	}
	metricsList := public.MetricsRangeQueryList{
		Kind: "MetricsRangeQueryList",
		Id:   kafkaId,
	}
	metrics, err := presenters.PresentMetricsByRangeQuery(kafkaMetrics)
	if err != nil {
		glog.Error("An error occurred while attempting to present metrics data ", err.Error())
		return
	}
	metricsList.Items = metrics
	output, marshalErr := json.MarshalIndent(metricsList, "", "    ")
	if marshalErr != nil {
		glog.Fatalf("Failed to format metrics list: %s", err.Error())
	}

	glog.V(10).Infof("%s %s", kafkaId, output)

}
