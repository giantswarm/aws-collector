package collector

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/service/support"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"

	"github.com/giantswarm/aws-collector/client/aws"
	"github.com/giantswarm/aws-collector/service/internal/cache"
)

const (
	// categoryServiceLimit is the category returned by Trusted Advisor for checks
	// related to service limits and usage.
	categoryServiceLimit = "service_limits"
)

const (
	indexRegion  = 0
	indexService = 1
	indexName    = 2
	indexLimit   = 3
	indexUsage   = 4
)

const (
	// resourceMetadataLength is the length of resource metadata we expect.
	resourceMetadataLength = 6
)

const (
	labelRegion  = "region"
	labelService = "service"
)

const (
	// __TrustedAdvisorCache__ is used as temporal cache key to save TrustedAdvisor response.
	prefixTrustedAdvisorcacheKey = "__TrustedAdvisorCache__"
)

var (
	getChecksDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "trusted_advisor_get_checks_duration",
		Help:      "Histogram for the duration of Trusted Advisor get checks calls.",
	})
	getResourcesDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "trusted_advisor_get_resources_duration",
		Help:      "Histogram for the duration of Trusted Advisor get resource calls.",
	})
	serviceLimit *prometheus.Desc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "service_limit"),
		"Service limits as reported by Trusted Advisor.",
		[]string{
			labelAccountID,
			labelRegion,
			labelService,
			labelName,
		},
		nil,
	)
	serviceUsage *prometheus.Desc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "service_usage"),
		"Service usage as reported by Trusted Advisor.",
		[]string{
			labelAccountID,
			labelRegion,
			labelService,
			labelName,
		},
		nil,
	)
)

type TrustedAdvisorConfig struct {
	Helper *helper
	Logger micrologger.Logger
}

type TrustedAdvisor struct {
	cache  *trustedAdvisorCache
	helper *helper
	logger micrologger.Logger
}

type trustedAdvisorCache struct {
	cache *cache.StringCache
}

type trustedAdvisorInfoResponse struct {
	TrustedAdvisors []trustedAdvisorInfo
}

type trustedAdvisorInfo struct {
	AccountID string
	CheckID   *string
	Resources []trustedAdvisorResource
}

type trustedAdvisorResource struct {
	Region    *string
	Service   *string
	LimitName *string
	Limit     *string
	Usage     *string
}

func NewTrustedAdvisor(config TrustedAdvisorConfig) (*TrustedAdvisor, error) {
	if config.Helper == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Helper must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	t := &TrustedAdvisor{
		cache:  newTrustedAdvisorCache(time.Minute * 5),
		helper: config.Helper,
		logger: config.Logger,
	}

	return t, nil
}

func newTrustedAdvisorCache(expiration time.Duration) *trustedAdvisorCache {
	cache := &trustedAdvisorCache{
		cache: cache.NewStringCache(expiration),
	}

	return cache
}

func (n *trustedAdvisorCache) Get(key string) (*trustedAdvisorInfoResponse, error) {
	var c trustedAdvisorInfoResponse
	raw, exists := n.cache.Get(getTrustedAdvisorCacheKey(key))
	if exists {
		err := json.Unmarshal(raw, &c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return &c, nil
}

func (n *trustedAdvisorCache) Set(key string, content trustedAdvisorInfoResponse) error {
	contentSerialized, err := json.Marshal(content)
	if err != nil {
		return microerror.Mask(err)
	}

	n.cache.Set(getTrustedAdvisorCacheKey(key), contentSerialized)

	return nil
}

func getTrustedAdvisorCacheKey(key string) string {
	return prefixTrustedAdvisorcacheKey + key
}

func (t *TrustedAdvisor) Collect(ch chan<- prometheus.Metric) error {
	reconciledClusters, err := t.helper.ListReconciledClusters()
	if err != nil {
		return microerror.Mask(err)
	}

	awsClientsList, err := t.helper.GetAWSClients(context.Background(), reconciledClusters)
	if err != nil {
		return microerror.Mask(err)
	}

	var g errgroup.Group

	for _, item := range awsClientsList {
		awsClients := item

		g.Go(func() error {
			err := t.collectForAccount(ch, awsClients)
			if err != nil {
				return microerror.Mask(err)
			}

			return nil
		})
	}

	err = g.Wait()
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (t *TrustedAdvisor) Describe(ch chan<- *prometheus.Desc) error {
	ch <- serviceLimit
	ch <- serviceUsage
	return nil
}

func (t *TrustedAdvisor) collectForAccount(ch chan<- prometheus.Metric, awsClients aws.Clients) error {
	accountID, err := t.helper.AWSAccountID(awsClients)
	if err != nil {
		return microerror.Mask(err)
	}
	var trustedAdvisorInfo *trustedAdvisorInfoResponse
	// Check if response is cached
	trustedAdvisorInfo, err = t.cache.Get(accountID)
	if err != nil {
		return microerror.Mask(err)
	}

	//Cache empty, getting from API
	if trustedAdvisorInfo == nil || trustedAdvisorInfo.trustedAdvisors == nil {
		trustedAdvisorInfo, err = t.getTrustedAdvisorInfoFromAPI(accountID, awsClients)
		if err != nil {
			return microerror.Mask(err)
		}
		if trustedAdvisorInfo != nil {
			err = t.cache.Set(accountID, *trustedAdvisorInfo)
			if err != nil {
				return microerror.Mask(err)
			}
		}
	}
	if trustedAdvisorInfo != nil {
		for _, ta := range trustedAdvisorInfo.trustedAdvisors {

			for _, resource := range ta.Resources {
				limit, usage, err := resourceToMetrics(resource, accountID)
				if err != nil {
					return microerror.Mask(err)
				}
				ch <- limit
				ch <- usage
			}
		}
	}

	return nil
}

// getTrustedAdvisorInfoFromAPI collects Trused Advisor Info from AWS API
func (t *TrustedAdvisor) getTrustedAdvisorInfoFromAPI(accountID string, awsClients aws.Clients) (*trustedAdvisorInfoResponse, error) {
	var res trustedAdvisorInfoResponse

	checks, err := t.getTrustedAdvisorChecks(awsClients)
	if IsUnsupportedPlan(err) {
		// While iterating through all kinds of account related AWS clients, we may
		// or may not be able to work against the Trusted Advisor API, depending on
		// the account's support plans.
		return nil, nil
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	var trustedAdvisors []trustedAdvisorInfo
	{
		var g errgroup.Group

		for _, check := range checks {
			// Ignore any checks that don't relate to service limits.
			if *check.Category != categoryServiceLimit {
				continue
			}
			// Register the check ID for the current loop scope so it can safely be used
			// in the goroutine below, which is execute in parallel.
			id := check.Id

			g.Go(func() error {
				var TAresources []trustedAdvisorResource
				{
					resources, err := t.getTrustedAdvisorResources(id, awsClients)
					if err != nil {
						return microerror.Mask(err)
					}
					for _, resource := range resources {
						// One Trusted Advisor check returns the nil string for current usage.
						// Skip it.
						if len(resource.Metadata) == 6 && resource.Metadata[4] == nil {
							continue
						}
						if len(resource.Metadata) != resourceMetadataLength {
							return invalidResourceError
						}
						TAResource := trustedAdvisorResource{
							Region:    resource.Metadata[indexRegion],
							Service:   resource.Metadata[indexService],
							LimitName: resource.Metadata[indexName],
							Limit:     resource.Metadata[indexLimit],
							Usage:     resource.Metadata[indexUsage],
						}
						TAresources = append(TAresources, TAResource)
					}
				}
				trustedAdvisor := trustedAdvisorInfo{
					AccountID: accountID,
					CheckID:   id,
					Resources: TAresources,
				}
				trustedAdvisors = append(trustedAdvisors, trustedAdvisor)

				return nil
			})
		}

		err = g.Wait()
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}
	res.trustedAdvisors = trustedAdvisors
	return &res, nil
}

// getTrustedAdvisorCheckDescriptions calls Trusted Advisor API to get all
// available checks.
func (t *TrustedAdvisor) getTrustedAdvisorChecks(awsClients aws.Clients) ([]*support.TrustedAdvisorCheckDescription, error) {
	timer := prometheus.NewTimer(getChecksDuration)

	englishLanguage := "en"
	describeChecksInput := &support.DescribeTrustedAdvisorChecksInput{
		Language: &englishLanguage,
	}
	describeChecksOutput, err := awsClients.Support.DescribeTrustedAdvisorChecks(describeChecksInput)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	timer.ObserveDuration()

	return describeChecksOutput.Checks, nil
}

// getTrustedAdvisorResources calls Trusted Advisor API to get flagged resources
// of the given check ID.
func (t *TrustedAdvisor) getTrustedAdvisorResources(id *string, awsClients aws.Clients) ([]*support.TrustedAdvisorResourceDetail, error) {
	timer := prometheus.NewTimer(getResourcesDuration)

	checkResultInput := &support.DescribeTrustedAdvisorCheckResultInput{
		CheckId: id,
	}
	checkResultOutput, err := awsClients.Support.DescribeTrustedAdvisorCheckResult(checkResultInput)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	timer.ObserveDuration()

	return checkResultOutput.Result.FlaggedResources, nil
}

func resourceToMetrics(resource trustedAdvisorResource, accountID string) (prometheus.Metric, prometheus.Metric, error) {
	region := resource.Region
	service := resource.Service
	limitName := resource.LimitName

	limit := resource.Limit
	usage := resource.Usage

	if limit == nil {
		return nil, nil, nilLimitError
	}
	if usage == nil {
		return nil, nil, nilUsageError
	}

	limitInt, err := strconv.Atoi(*limit)
	if err != nil {
		return nil, nil, microerror.Mask(err)
	}

	usageInt, err := strconv.Atoi(*usage)
	if err != nil {
		return nil, nil, microerror.Mask(err)
	}

	limitMetric := prometheus.MustNewConstMetric(
		serviceLimit, prometheus.GaugeValue, float64(limitInt), accountID, *region, *service, *limitName,
	)
	usageMetric := prometheus.MustNewConstMetric(
		serviceUsage, prometheus.GaugeValue, float64(usageInt), accountID, *region, *service, *limitName,
	)

	return limitMetric, usageMetric, nil
}
