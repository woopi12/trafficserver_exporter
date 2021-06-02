package main

import (
	//"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"

	//"io/ioutil"
	"net/http"
	"net/url"

	//"os"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	up = prometheus.NewDesc(
		"trafficserver_up",
		"Was talking to Trafficserver successfully",
		nil, nil,
	)
	invalidChars = regexp.MustCompile("[^a-zA-Z0-9:_]")
)

type TrafficServerCollector struct {
	trafficServerScrapeUri     string
	trafficServerScrapeTimeout int
	trafficServerSslVerify     bool
}

type Metrics struct {
	Counters Counters `json:"global"`
}

// Very incomplete list of counters, but these are the ones we know we care
// about right now. As we categorize and sort the metrics more, we'll bring
// more Counters and Gauges over from the structs folder.
type Counters struct {
	Proxy_node_restarts_proxy_stop_time                                float64 `json:"proxy.node.restarts.proxy.stop_time,string"`
	Proxy_node_restarts_proxy_restart_count                            float64 `json:"proxy.node.restarts.proxy.restart_count,string"`
	Proxy_process_http_user_agent_total_response_bytes                 float64 `json:"proxy.process.http.user_agent_total_response_bytes,string"`
	Proxy_process_http_origin_server_total_request_bytes               float64 `json:"proxy.process.http.origin_server_total_request_bytes,string"`
	Proxy_process_http_origin_server_total_response_bytes              float64 `json:"proxy.process.http.origin_server_total_response_bytes,string"`
	Proxy_process_user_agent_total_bytes                               float64 `json:"proxy.process.user_agent_total_bytes,string"`
	Proxy_process_origin_server_total_bytes                            float64 `json:"proxy.process.origin_server_total_bytes,string"`
	Proxy_process_cache_total_hits                                     float64 `json:"proxy.process.cache_total_hits,string"`
	Proxy_process_cache_total_misses                                   float64 `json:"proxy.process.cache_total_misses,string"`
	Proxy_process_cache_total_requests                                 float64 `json:"proxy.process.cache_total_requests,string"`
	Proxy_process_cache_total_hits_bytes                               float64 `json:"proxy.process.cache_total_hits_bytes,string"`
	Proxy_process_cache_total_misses_bytes                             float64 `json:"proxy.process.cache_total_misses_bytes,string"`
	Proxy_process_cache_total_bytes                                    float64 `json:"proxy.process.cache_total_bytes,string"`
	Proxy_process_current_server_connections                           float64 `json:"proxy.process.current_server_connections,string"`
	Proxy_node_proxy_running                                           float64 `json:"proxy.node.proxy_running,string"`
	Proxy_process_http_user_agent_total_request_bytes                  float64 `json:"proxy.process.http.user_agent_total_request_bytes,string"`
	Proxy_process_http_completed_requests                              float64 `json:"proxy.process.http.completed_requests,string"`
	Proxy_process_http_total_incoming_connections                      float64 `json:"proxy.process.http.total_incoming_connections,string"`
	Proxy_process_http_total_client_connections                        float64 `json:"proxy.process.http.total_client_connections,string"`
	Proxy_process_http_total_client_connections_ipv4                   float64 `json:"proxy.process.http.total_client_connections_ipv4,string"`
	Proxy_process_http_total_client_connections_ipv6                   float64 `json:"proxy.process.http.total_client_connections_ipv6,string"`
	Proxy_process_http_total_server_connections                        float64 `json:"proxy.process.http.total_server_connections,string"`
	Proxy_process_http_total_parent_proxy_connections                  float64 `json:"proxy.process.http.total_parent_proxy_connections,string"`
	Proxy_process_http_total_parent_retries                            float64 `json:"proxy.process.http.total_parent_retries,string"`
	Proxy_process_http_total_parent_switches                           float64 `json:"proxy.process.http.total_parent_switches,string"`
	Proxy_process_http_total_parent_retries_exhausted                  float64 `json:"proxy.process.http.total_parent_retries_exhausted,string"`
	Proxy_process_http_total_parent_marked_down_count                  float64 `json:"proxy.process.http.total_parent_marked_down_count,string"`
	Proxy_process_http_avg_transactions_per_client_connection          float64 `json:"proxy.process.http.avg_transactions_per_client_connection,string"`
	Proxy_process_http_avg_transactions_per_server_connection          float64 `json:"proxy.process.http.avg_transactions_per_server_connection,string"`
	Proxy_process_http_transaction_counts_errors_pre_accept_hangups    float64 `json:"proxy.process.http.transaction_counts.errors.pre_accept_hangups,string"`
	Proxy_process_http_transaction_totaltime_errors_pre_accept_hangups float64 `json:"proxy.process.http.transaction_totaltime.errors.pre_accept_hangups,string"`
	Proxy_process_http_incoming_requests                               float64 `json:"proxy.process.http.incoming_requests,string"`
	Proxy_process_http_outgoing_requests                               float64 `json:"proxy.process.http.outgoing_requests,string"`
	Proxy_process_http_incoming_responses                              float64 `json:"proxy.process.http.incoming_responses,string"`
	Proxy_process_http_invalid_client_requests                         float64 `json:"proxy.process.http.invalid_client_requests,string"`
	Proxy_process_http_missing_host_hdr                                float64 `json:"proxy.process.http.missing_host_hdr,string"`
	Proxy_process_http_get_requests                                    float64 `json:"proxy.process.http.get_requests,string"`
	Proxy_process_http_head_requests                                   float64 `json:"proxy.process.http.head_requests,string"`
	Proxy_process_http_trace_requests                                  float64 `json:"proxy.process.http.trace_requests,string"`
	Proxy_process_http_options_requests                                float64 `json:"proxy.process.http.options_requests,string"`
	Proxy_process_http_post_requests                                   float64 `json:"proxy.process.http.post_requests,string"`
	Proxy_process_http_put_requests                                    float64 `json:"proxy.process.http.put_requests,string"`
	Proxy_process_http_push_requests                                   float64 `json:"proxy.process.http.push_requests,string"`
	Proxy_process_http_delete_requests                                 float64 `json:"proxy.process.http.delete_requests,string"`
	Proxy_process_http_purge_requests                                  float64 `json:"proxy.process.http.purge_requests,string"`
	Proxy_process_http_connect_requests                                float64 `json:"proxy.process.http.connect_requests,string"`
	Proxy_process_http_extension_method_requests                       float64 `json:"proxy.process.http.extension_method_requests,string"`
	Proxy_process_http_broken_server_connections                       float64 `json:"proxy.process.http.broken_server_connections,string"`
	Proxy_process_http_cache_lookups                                   float64 `json:"proxy.process.http.cache_lookups,string"`
	Proxy_process_http_cache_writes                                    float64 `json:"proxy.process.http.cache_writes,string"`
	Proxy_process_http_cache_updates                                   float64 `json:"proxy.process.http.cache_updates,string"`
	Proxy_process_http_cache_deletes                                   float64 `json:"proxy.process.http.cache_deletes,string"`
	Proxy_process_http_tunnels                                         float64 `json:"proxy.process.http.tunnels,string"`
	Proxy_process_http_parent_proxy_transaction_time                   float64 `json:"proxy.process.http.parent_proxy_transaction_time,string"`
	Proxy_process_http_user_agent_request_header_total_size            float64 `json:"proxy.process.http.user_agent_request_header_total_size,string"`
	Proxy_process_http_user_agent_response_header_total_size           float64 `json:"proxy.process.http.user_agent_response_header_total_size,string"`
	Proxy_process_http_user_agent_request_document_total_size          float64 `json:"proxy.process.http.user_agent_request_document_total_size,string"`
	Proxy_process_http_user_agent_response_document_total_size         float64 `json:"proxy.process.http.user_agent_response_document_total_size,string"`
	Proxy_process_http_origin_server_request_header_total_size         float64 `json:"proxy.process.http.origin_server_request_header_total_size,string"`
	Proxy_process_http_origin_server_response_header_total_size        float64 `json:"proxy.process.http.origin_server_response_header_total_size,string"`
	Proxy_process_http_origin_server_request_document_total_size       float64 `json:"proxy.process.http.origin_server_request_document_total_size,string"`
	Proxy_process_http_origin_server_response_document_total_size      float64 `json:"proxy.process.http.origin_server_response_document_total_size,string"`
	Proxy_process_http_parent_proxy_request_total_bytes                float64 `json:"proxy.process.http.parent_proxy_request_total_bytes,string"`
	Proxy_process_http_parent_proxy_response_total_bytes               float64 `json:"proxy.process.http.parent_proxy_response_total_bytes,string"`
	Proxy_process_http_pushed_response_header_total_size               float64 `json:"proxy.process.http.pushed_response_header_total_size,string"`
	Proxy_process_http_pushed_document_total_size                      float64 `json:"proxy.process.http.pushed_document_total_size,string"`
	Proxy_process_http_response_document_size_100                      float64 `json:"proxy.process.http.response_document_size_100,string"`
	Proxy_process_http_response_document_size_1K                       float64 `json:"proxy.process.http.response_document_size_1K,string"`
	Proxy_process_http_response_document_size_3K                       float64 `json:"proxy.process.http.response_document_size_3K,string"`
	Proxy_process_http_response_document_size_5K                       float64 `json:"proxy.process.http.response_document_size_5K,string"`
	Proxy_process_http_response_document_size_10K                      float64 `json:"proxy.process.http.response_document_size_10K,string"`
	Proxy_process_http_response_document_size_1M                       float64 `json:"proxy.process.http.response_document_size_1M,string"`
	Proxy_process_http_response_document_size_inf                      float64 `json:"proxy.process.http.response_document_size_inf,string"`
	Proxy_process_http_request_document_size_100                       float64 `json:"proxy.process.http.request_document_size_100,string"`
	Proxy_process_http_request_document_size_1K                        float64 `json:"proxy.process.http.request_document_size_1K,string"`
	Proxy_process_http_request_document_size_3K                        float64 `json:"proxy.process.http.request_document_size_3K,string"`
	Proxy_process_http_request_document_size_5K                        float64 `json:"proxy.process.http.request_document_size_5K,string"`
	Proxy_process_http_request_document_size_10K                       float64 `json:"proxy.process.http.request_document_size_10K,string"`
	Proxy_process_http_request_document_size_1M                        float64 `json:"proxy.process.http.request_document_size_1M,string"`
	Proxy_process_http_request_document_size_inf                       float64 `json:"proxy.process.http.request_document_size_inf,string"`
	Proxy_process_http_user_agent_speed_bytes_per_sec_100              float64 `json:"proxy.process.http.user_agent_speed_bytes_per_sec_100,string"`
	Proxy_process_http_user_agent_speed_bytes_per_sec_1K               float64 `json:"proxy.process.http.user_agent_speed_bytes_per_sec_1K,string"`
	Proxy_process_http_user_agent_speed_bytes_per_sec_10K              float64 `json:"proxy.process.http.user_agent_speed_bytes_per_sec_10K,string"`
	Proxy_process_http_user_agent_speed_bytes_per_sec_100K             float64 `json:"proxy.process.http.user_agent_speed_bytes_per_sec_100K,string"`
	Proxy_process_http_user_agent_speed_bytes_per_sec_1M               float64 `json:"proxy.process.http.user_agent_speed_bytes_per_sec_1M,string"`
	Proxy_process_http_user_agent_speed_bytes_per_sec_10M              float64 `json:"proxy.process.http.user_agent_speed_bytes_per_sec_10M,string"`
	Proxy_process_http_user_agent_speed_bytes_per_sec_100M             float64 `json:"proxy.process.http.user_agent_speed_bytes_per_sec_100M,string"`
	Proxy_process_http_origin_server_speed_bytes_per_sec_100           float64 `json:"proxy.process.http.origin_server_speed_bytes_per_sec_100,string"`
	Proxy_process_http_origin_server_speed_bytes_per_sec_1K            float64 `json:"proxy.process.http.origin_server_speed_bytes_per_sec_1K,string"`
	Proxy_process_http_origin_server_speed_bytes_per_sec_10K           float64 `json:"proxy.process.http.origin_server_speed_bytes_per_sec_10K,string"`
	Proxy_process_http_origin_server_speed_bytes_per_sec_100K          float64 `json:"proxy.process.http.origin_server_speed_bytes_per_sec_100K,string"`
	Proxy_process_http_origin_server_speed_bytes_per_sec_1M            float64 `json:"proxy.process.http.origin_server_speed_bytes_per_sec_1M,string"`
	Proxy_process_http_origin_server_speed_bytes_per_sec_10M           float64 `json:"proxy.process.http.origin_server_speed_bytes_per_sec_10M,string"`
	Proxy_process_http_origin_server_speed_bytes_per_sec_100M          float64 `json:"proxy.process.http.origin_server_speed_bytes_per_sec_100M,string"`
	Proxy_process_http_cache_hit_fresh                                 float64 `json:"proxy.process.http.cache_hit_fresh,string"`
	Proxy_process_http_cache_hit_mem_fresh                             float64 `json:"proxy.process.http.cache_hit_mem_fresh,string"`
	Proxy_process_http_cache_hit_revalidated                           float64 `json:"proxy.process.http.cache_hit_revalidated,string"`
	Proxy_process_http_cache_hit_ims                                   float64 `json:"proxy.process.http.cache_hit_ims,string"`
	Proxy_process_http_cache_hit_stale_served                          float64 `json:"proxy.process.http.cache_hit_stale_served,string"`
	Proxy_process_http_cache_miss_cold                                 float64 `json:"proxy.process.http.cache_miss_cold,string"`
	Proxy_process_http_cache_miss_changed                              float64 `json:"proxy.process.http.cache_miss_changed,string"`
	Proxy_process_http_cache_miss_client_no_cache                      float64 `json:"proxy.process.http.cache_miss_client_no_cache,string"`
	Proxy_process_http_cache_miss_client_not_cacheable                 float64 `json:"proxy.process.http.cache_miss_client_not_cacheable,string"`
	Proxy_process_http_cache_miss_ims                                  float64 `json:"proxy.process.http.cache_miss_ims,string"`
	Proxy_process_http_cache_read_error                                float64 `json:"proxy.process.http.cache_read_error,string"`
	Proxy_process_http_tcp_hit_count_stat                              float64 `json:"proxy.process.http.tcp_hit_count_stat,string"`
	Proxy_process_http_tcp_hit_user_agent_bytes_stat                   float64 `json:"proxy.process.http.tcp_hit_user_agent_bytes_stat,string"`
	Proxy_process_http_tcp_hit_origin_server_bytes_stat                float64 `json:"proxy.process.http.tcp_hit_origin_server_bytes_stat,string"`
	Proxy_process_http_tcp_miss_count_stat                             float64 `json:"proxy.process.http.tcp_miss_count_stat,string"`
	Proxy_process_http_tcp_miss_user_agent_bytes_stat                  float64 `json:"proxy.process.http.tcp_miss_user_agent_bytes_stat,string"`
	Proxy_process_http_tcp_miss_origin_server_bytes_stat               float64 `json:"proxy.process.http.tcp_miss_origin_server_bytes_stat,string"`
	Proxy_process_http_tcp_expired_miss_count_stat                     float64 `json:"proxy.process.http.tcp_expired_miss_count_stat,string"`
	Proxy_process_http_tcp_expired_miss_user_agent_bytes_stat          float64 `json:"proxy.process.http.tcp_expired_miss_user_agent_bytes_stat,string"`
	Proxy_process_http_tcp_expired_miss_origin_server_bytes_stat       float64 `json:"proxy.process.http.tcp_expired_miss_origin_server_bytes_stat,string"`
	Proxy_process_http_tcp_refresh_hit_count_stat                      float64 `json:"proxy.process.http.tcp_refresh_hit_count_stat,string"`
	Proxy_process_http_tcp_refresh_hit_user_agent_bytes_stat           float64 `json:"proxy.process.http.tcp_refresh_hit_user_agent_bytes_stat,string"`
	Proxy_process_http_tcp_refresh_hit_origin_server_bytes_stat        float64 `json:"proxy.process.http.tcp_refresh_hit_origin_server_bytes_stat,string"`
	Proxy_process_http_tcp_refresh_miss_count_stat                     float64 `json:"proxy.process.http.tcp_refresh_miss_count_stat,string"`
	Proxy_process_http_tcp_refresh_miss_user_agent_bytes_stat          float64 `json:"proxy.process.http.tcp_refresh_miss_user_agent_bytes_stat,string"`
	Proxy_process_http_tcp_refresh_miss_origin_server_bytes_stat       float64 `json:"proxy.process.http.tcp_refresh_miss_origin_server_bytes_stat,string"`
	Proxy_process_http_tcp_client_refresh_count_stat                   float64 `json:"proxy.process.http.tcp_client_refresh_count_stat,string"`
	Proxy_process_http_tcp_client_refresh_user_agent_bytes_stat        float64 `json:"proxy.process.http.tcp_client_refresh_user_agent_bytes_stat,string"`
	Proxy_process_http_tcp_client_refresh_origin_server_bytes_stat     float64 `json:"proxy.process.http.tcp_client_refresh_origin_server_bytes_stat,string"`
	Proxy_process_http_tcp_ims_hit_count_stat                          float64 `json:"proxy.process.http.tcp_ims_hit_count_stat,string"`
	Proxy_process_http_tcp_ims_hit_user_agent_bytes_stat               float64 `json:"proxy.process.http.tcp_ims_hit_user_agent_bytes_stat,string"`
	Proxy_process_http_tcp_ims_hit_origin_server_bytes_stat            float64 `json:"proxy.process.http.tcp_ims_hit_origin_server_bytes_stat,string"`
	Proxy_process_http_tcp_ims_miss_count_stat                         float64 `json:"proxy.process.http.tcp_ims_miss_count_stat,string"`
	Proxy_process_http_tcp_ims_miss_user_agent_bytes_stat              float64 `json:"proxy.process.http.tcp_ims_miss_user_agent_bytes_stat,string"`
	Proxy_process_http_tcp_ims_miss_origin_server_bytes_stat           float64 `json:"proxy.process.http.tcp_ims_miss_origin_server_bytes_stat,string"`
	Proxy_process_http_err_client_abort_count_stat                     float64 `json:"proxy.process.http.err_client_abort_count_stat,string"`
	Proxy_process_http_err_client_abort_user_agent_bytes_stat          float64 `json:"proxy.process.http.err_client_abort_user_agent_bytes_stat,string"`
	Proxy_process_http_err_client_abort_origin_server_bytes_stat       float64 `json:"proxy.process.http.err_client_abort_origin_server_bytes_stat,string"`
	Proxy_process_http_err_client_read_error_count_stat                float64 `json:"proxy.process.http.err_client_read_error_count_stat,string"`
	Proxy_process_http_err_client_read_error_user_agent_bytes_stat     float64 `json:"proxy.process.http.err_client_read_error_user_agent_bytes_stat,string"`
	Proxy_process_http_err_client_read_error_origin_server_bytes_stat  float64 `json:"proxy.process.http.err_client_read_error_origin_server_bytes_stat,string"`
	Proxy_process_http_err_connect_fail_count_stat                     float64 `json:"proxy.process.http.err_connect_fail_count_stat,string"`
	Proxy_process_http_err_connect_fail_user_agent_bytes_stat          float64 `json:"proxy.process.http.err_connect_fail_user_agent_bytes_stat,string"`
	Proxy_process_http_err_connect_fail_origin_server_bytes_stat       float64 `json:"proxy.process.http.err_connect_fail_origin_server_bytes_stat,string"`
	Proxy_process_http_misc_count_stat                                 float64 `json:"proxy.process.http.misc_count_stat,string"`
	Proxy_process_http_misc_user_agent_bytes_stat                      float64 `json:"proxy.process.http.misc_user_agent_bytes_stat,string"`
	Proxy_process_http_http_misc_origin_server_bytes_stat              float64 `json:"proxy.process.http.http_misc_origin_server_bytes_stat,string"`
	Proxy_process_http_background_fill_bytes_aborted_stat              float64 `json:"proxy.process.http.background_fill_bytes_aborted_stat,string"`
	Proxy_process_http_background_fill_bytes_completed_stat            float64 `json:"proxy.process.http.background_fill_bytes_completed_stat,string"`
	Proxy_process_http_cache_write_errors                              float64 `json:"proxy.process.http.cache_write_errors,string"`
	Proxy_process_http_cache_read_errors                               float64 `json:"proxy.process.http.cache_read_errors,string"`
	Proxy_process_http_100_responses                                   float64 `json:"proxy.process.http.100_responses,string"`
	Proxy_process_http_101_responses                                   float64 `json:"proxy.process.http.101_responses,string"`
	Proxy_process_http_1xx_responses                                   float64 `json:"proxy.process.http.1xx_responses,string"`
	Proxy_process_http_200_responses                                   float64 `json:"proxy.process.http.200_responses,string"`
	Proxy_process_http_201_responses                                   float64 `json:"proxy.process.http.201_responses,string"`
	Proxy_process_http_202_responses                                   float64 `json:"proxy.process.http.202_responses,string"`
	Proxy_process_http_203_responses                                   float64 `json:"proxy.process.http.203_responses,string"`
	Proxy_process_http_204_responses                                   float64 `json:"proxy.process.http.204_responses,string"`
	Proxy_process_http_205_responses                                   float64 `json:"proxy.process.http.205_responses,string"`
	Proxy_process_http_206_responses                                   float64 `json:"proxy.process.http.206_responses,string"`
	Proxy_process_http_2xx_responses                                   float64 `json:"proxy.process.http.2xx_responses,string"`
	Proxy_process_http_300_responses                                   float64 `json:"proxy.process.http.300_responses,string"`
	Proxy_process_http_301_responses                                   float64 `json:"proxy.process.http.301_responses,string"`
	Proxy_process_http_302_responses                                   float64 `json:"proxy.process.http.302_responses,string"`
	Proxy_process_http_303_responses                                   float64 `json:"proxy.process.http.303_responses,string"`
	Proxy_process_http_304_responses                                   float64 `json:"proxy.process.http.304_responses,string"`
	Proxy_process_http_305_responses                                   float64 `json:"proxy.process.http.305_responses,string"`
	Proxy_process_http_307_responses                                   float64 `json:"proxy.process.http.307_responses,string"`
	Proxy_process_http_308_responses                                   float64 `json:"proxy.process.http.308_responses,string"`
	Proxy_process_http_3xx_responses                                   float64 `json:"proxy.process.http.3xx_responses,string"`
	Proxy_process_http_400_responses                                   float64 `json:"proxy.process.http.400_responses,string"`
	Proxy_process_http_401_responses                                   float64 `json:"proxy.process.http.401_responses,string"`
	Proxy_process_http_402_responses                                   float64 `json:"proxy.process.http.402_responses,string"`
	Proxy_process_http_403_responses                                   float64 `json:"proxy.process.http.403_responses,string"`
	Proxy_process_http_404_responses                                   float64 `json:"proxy.process.http.404_responses,string"`
	Proxy_process_http_405_responses                                   float64 `json:"proxy.process.http.405_responses,string"`
	Proxy_process_http_406_responses                                   float64 `json:"proxy.process.http.406_responses,string"`
	Proxy_process_http_407_responses                                   float64 `json:"proxy.process.http.407_responses,string"`
	Proxy_process_http_408_responses                                   float64 `json:"proxy.process.http.408_responses,string"`
	Proxy_process_http_409_responses                                   float64 `json:"proxy.process.http.409_responses,string"`
	Proxy_process_http_410_responses                                   float64 `json:"proxy.process.http.410_responses,string"`
	Proxy_process_http_411_responses                                   float64 `json:"proxy.process.http.411_responses,string"`
	Proxy_process_http_412_responses                                   float64 `json:"proxy.process.http.412_responses,string"`
	Proxy_process_http_413_responses                                   float64 `json:"proxy.process.http.413_responses,string"`
	Proxy_process_http_414_responses                                   float64 `json:"proxy.process.http.414_responses,string"`
	Proxy_process_http_415_responses                                   float64 `json:"proxy.process.http.415_responses,string"`
	Proxy_process_http_416_responses                                   float64 `json:"proxy.process.http.416_responses,string"`
	Proxy_process_http_4xx_responses                                   float64 `json:"proxy.process.http.4xx_responses,string"`
	Proxy_process_http_500_responses                                   float64 `json:"proxy.process.http.500_responses,string"`
	Proxy_process_http_501_responses                                   float64 `json:"proxy.process.http.501_responses,string"`
	Proxy_process_http_502_responses                                   float64 `json:"proxy.process.http.502_responses,string"`
	Proxy_process_http_503_responses                                   float64 `json:"proxy.process.http.503_responses,string"`
	Proxy_process_http_504_responses                                   float64 `json:"proxy.process.http.504_responses,string"`
	Proxy_process_http_505_responses                                   float64 `json:"proxy.process.http.505_responses,string"`
	Proxy_process_http_5xx_responses                                   float64 `json:"proxy.process.http.5xx_responses,string"`
	Proxy_process_http_transaction_counts_hit_fresh                    float64 `json:"proxy.process.http.transaction_counts.hit_fresh,string"`
	Proxy_process_http_transaction_totaltime_hit_fresh                 float64 `json:"proxy.process.http.transaction_totaltime.hit_fresh,string"`
	Proxy_process_http_transaction_counts_hit_fresh_process            float64 `json:"proxy.process.http.transaction_counts.hit_fresh.process,string"`
	Proxy_process_http_transaction_totaltime_hit_fresh_process         float64 `json:"proxy.process.http.transaction_totaltime.hit_fresh.process,string"`
	Proxy_process_http_transaction_counts_hit_revalidated              float64 `json:"proxy.process.http.transaction_counts.hit_revalidated,string"`
	Proxy_process_http_transaction_totaltime_hit_revalidated           float64 `json:"proxy.process.http.transaction_totaltime.hit_revalidated,string"`
	Proxy_process_http_transaction_counts_miss_cold                    float64 `json:"proxy.process.http.transaction_counts.miss_cold,string"`
	Proxy_process_http_transaction_totaltime_miss_cold                 float64 `json:"proxy.process.http.transaction_totaltime.miss_cold,string"`
	Proxy_process_http_transaction_counts_miss_not_cacheable           float64 `json:"proxy.process.http.transaction_counts.miss_not_cacheable,string"`
	Proxy_process_http_transaction_totaltime_miss_not_cacheable        float64 `json:"proxy.process.http.transaction_totaltime.miss_not_cacheable,string"`
	Proxy_process_http_transaction_counts_miss_changed                 float64 `json:"proxy.process.http.transaction_counts.miss_changed,string"`
	Proxy_process_http_transaction_totaltime_miss_changed              float64 `json:"proxy.process.http.transaction_totaltime.miss_changed,string"`
	Proxy_process_http_transaction_counts_miss_client_no_cache         float64 `json:"proxy.process.http.transaction_counts.miss_client_no_cache,string"`
	Proxy_process_http_transaction_totaltime_miss_client_no_cache      float64 `json:"proxy.process.http.transaction_totaltime.miss_client_no_cache,string"`
	Proxy_process_http_transaction_counts_errors_aborts                float64 `json:"proxy.process.http.transaction_counts.errors.aborts,string"`
	Proxy_process_http_transaction_totaltime_errors_aborts             float64 `json:"proxy.process.http.transaction_totaltime.errors.aborts,string"`
	Proxy_process_http_transaction_counts_errors_possible_aborts       float64 `json:"proxy.process.http.transaction_counts.errors.possible_aborts,string"`
	Proxy_process_http_transaction_totaltime_errors_possible_aborts    float64 `json:"proxy.process.http.transaction_totaltime.errors.possible_aborts,string"`
	Proxy_process_http_transaction_counts_errors_connect_failed        float64 `json:"proxy.process.http.transaction_counts.errors.connect_failed,string"`
	Proxy_process_http_transaction_totaltime_errors_connect_failed     float64 `json:"proxy.process.http.transaction_totaltime.errors.connect_failed,string"`
	Proxy_process_http_transaction_counts_errors_other                 float64 `json:"proxy.process.http.transaction_counts.errors.other,string"`
	Proxy_process_http_transaction_totaltime_errors_other              float64 `json:"proxy.process.http.transaction_totaltime.errors.other,string"`
	Proxy_process_http_transaction_counts_other_unclassified           float64 `json:"proxy.process.http.transaction_counts.other.unclassified,string"`
	Proxy_process_http_transaction_totaltime_other_unclassified        float64 `json:"proxy.process.http.transaction_totaltime.other.unclassified,string"`
	Proxy_process_http_disallowed_post_100_continue                    float64 `json:"proxy.process.http.disallowed_post_100_continue,string"`
	Proxy_process_http_total_x_redirect_count                          float64 `json:"proxy.process.http.total_x_redirect_count,string"`
	Proxy_process_https_incoming_requests                              float64 `json:"proxy.process.https.incoming_requests,string"`
	Proxy_process_https_total_client_connections                       float64 `json:"proxy.process.https.total_client_connections,string"`
	Proxy_process_http_origin_connections_throttled_out                float64 `json:"proxy.process.http.origin_connections_throttled_out,string"`
	Proxy_process_http_post_body_too_large                             float64 `json:"proxy.process.http.post_body_too_large,string"`
	Proxy_process_http_milestone_ua_begin                              float64 `json:"proxy.process.http.milestone.ua_begin,string"`
	Proxy_process_http_milestone_ua_first_read                         float64 `json:"proxy.process.http.milestone.ua_first_read,string"`
	Proxy_process_http_milestone_ua_read_header_done                   float64 `json:"proxy.process.http.milestone.ua_read_header_done,string"`
	Proxy_process_http_milestone_ua_begin_write                        float64 `json:"proxy.process.http.milestone.ua_begin_write,string"`
	Proxy_process_http_milestone_ua_close                              float64 `json:"proxy.process.http.milestone.ua_close,string"`
	Proxy_process_http_milestone_server_first_connect                  float64 `json:"proxy.process.http.milestone.server_first_connect,string"`
	Proxy_process_http_milestone_server_connect                        float64 `json:"proxy.process.http.milestone.server_connect,string"`
	Proxy_process_http_milestone_server_connect_end                    float64 `json:"proxy.process.http.milestone.server_connect_end,string"`
	Proxy_process_http_milestone_server_begin_write                    float64 `json:"proxy.process.http.milestone.server_begin_write,string"`
	Proxy_process_http_milestone_server_first_read                     float64 `json:"proxy.process.http.milestone.server_first_read,string"`
	Proxy_process_http_milestone_server_read_header_done               float64 `json:"proxy.process.http.milestone.server_read_header_done,string"`
	Proxy_process_http_milestone_server_close                          float64 `json:"proxy.process.http.milestone.server_close,string"`
	Proxy_process_http_milestone_cache_open_read_begin                 float64 `json:"proxy.process.http.milestone.cache_open_read_begin,string"`
	Proxy_process_http_milestone_cache_open_read_end                   float64 `json:"proxy.process.http.milestone.cache_open_read_end,string"`
	Proxy_process_http_milestone_cache_open_write_begin                float64 `json:"proxy.process.http.milestone.cache_open_write_begin,string"`
	Proxy_process_http_milestone_cache_open_write_end                  float64 `json:"proxy.process.http.milestone.cache_open_write_end,string"`
	Proxy_process_http_milestone_dns_lookup_begin                      float64 `json:"proxy.process.http.milestone.dns_lookup_begin,string"`
	Proxy_process_http_milestone_dns_lookup_end                        float64 `json:"proxy.process.http.milestone.dns_lookup_end,string"`
	Proxy_process_http_milestone_sm_start                              float64 `json:"proxy.process.http.milestone.sm_start,string"`
	Proxy_process_http_milestone_sm_finish                             float64 `json:"proxy.process.http.milestone.sm_finish,string"`
	Proxy_process_net_calls_to_read                                    float64 `json:"proxy.process.net.calls_to_read,string"`
	Proxy_process_net_calls_to_read_nodata                             float64 `json:"proxy.process.net.calls_to_read_nodata,string"`
	Proxy_process_net_calls_to_readfromnet                             float64 `json:"proxy.process.net.calls_to_readfromnet,string"`
	Proxy_process_net_calls_to_readfromnet_afterpoll                   float64 `json:"proxy.process.net.calls_to_readfromnet_afterpoll,string"`
	Proxy_process_net_calls_to_write                                   float64 `json:"proxy.process.net.calls_to_write,string"`
	Proxy_process_net_calls_to_write_nodata                            float64 `json:"proxy.process.net.calls_to_write_nodata,string"`
	Proxy_process_net_calls_to_writetonet                              float64 `json:"proxy.process.net.calls_to_writetonet,string"`
	Proxy_process_net_calls_to_writetonet_afterpoll                    float64 `json:"proxy.process.net.calls_to_writetonet_afterpoll,string"`
	Proxy_process_net_inactivity_cop_lock_acquire_failure              float64 `json:"proxy.process.net.inactivity_cop_lock_acquire_failure,string"`
	Proxy_process_net_net_handler_run                                  float64 `json:"proxy.process.net.net_handler_run,string"`
	Proxy_process_net_read_bytes                                       float64 `json:"proxy.process.net.read_bytes,string"`
	Proxy_process_net_write_bytes                                      float64 `json:"proxy.process.net.write_bytes,string"`
	Proxy_process_net_fastopen_out_attempts                            float64 `json:"proxy.process.net.fastopen_out.attempts,string"`
	Proxy_process_net_fastopen_out_successes                           float64 `json:"proxy.process.net.fastopen_out.successes,string"`
	Proxy_process_socks_connections_successful                         float64 `json:"proxy.process.socks.connections_successful,string"`
	Proxy_process_socks_connections_unsuccessful                       float64 `json:"proxy.process.socks.connections_unsuccessful,string"`
	Proxy_process_net_connections_throttled_in                         float64 `json:"proxy.process.net.connections_throttled_in,string"`
	Proxy_process_net_connections_throttled_out                        float64 `json:"proxy.process.net.connections_throttled_out,string"`
	Proxy_process_net_max_requests_throttled_in                        float64 `json:"proxy.process.net.max.requests_throttled_in,string"`
	Proxy_process_cache_read_per_sec                                   float64 `json:"proxy.process.cache.read_per_sec,string"`
	Proxy_process_cache_write_per_sec                                  float64 `json:"proxy.process.cache.write_per_sec,string"`
	Proxy_process_cache_KB_read_per_sec                                float64 `json:"proxy.process.cache.KB_read_per_sec,string"`
	Proxy_process_cache_KB_write_per_sec                               float64 `json:"proxy.process.cache.KB_write_per_sec,string"`
	Proxy_process_hostdb_total_lookups                                 float64 `json:"proxy.process.hostdb.total_lookups,string"`
	Proxy_process_hostdb_total_hits                                    float64 `json:"proxy.process.hostdb.total_hits,string"`
	Proxy_process_hostdb_ttl                                           float64 `json:"proxy.process.hostdb.ttl,string"`
	Proxy_process_hostdb_ttl_expires                                   float64 `json:"proxy.process.hostdb.ttl_expires,string"`
	Proxy_process_hostdb_re_dns_on_reload                              float64 `json:"proxy.process.hostdb.re_dns_on_reload,string"`
	Proxy_process_hostdb_insert_duplicate_to_pending_dns               float64 `json:"proxy.process.hostdb.insert_duplicate_to_pending_dns,string"`
	Proxy_process_dns_total_dns_lookups                                float64 `json:"proxy.process.dns.total_dns_lookups,string"`
	Proxy_process_dns_lookup_avg_time                                  float64 `json:"proxy.process.dns.lookup_avg_time,string"`
	Proxy_process_dns_lookup_successes                                 float64 `json:"proxy.process.dns.lookup_successes,string"`
	Proxy_process_dns_fail_avg_time                                    float64 `json:"proxy.process.dns.fail_avg_time,string"`
	Proxy_process_dns_lookup_failures                                  float64 `json:"proxy.process.dns.lookup_failures,string"`
	Proxy_process_dns_retries                                          float64 `json:"proxy.process.dns.retries,string"`
	Proxy_process_dns_max_retries_exceeded                             float64 `json:"proxy.process.dns.max_retries_exceeded,string"`
	Proxy_process_http2_total_client_streams                           float64 `json:"proxy.process.http2.total_client_streams,string"`
	Proxy_process_http2_total_transactions_time                        float64 `json:"proxy.process.http2.total_transactions_time,string"`
	Proxy_process_http2_total_client_connections                       float64 `json:"proxy.process.http2.total_client_connections,string"`
	Proxy_process_http2_connection_errors                              float64 `json:"proxy.process.http2.connection_errors,string"`
	Proxy_process_http2_stream_errors                                  float64 `json:"proxy.process.http2.stream_errors,string"`
	Proxy_process_http2_session_die_default                            float64 `json:"proxy.process.http2.session_die_default,string"`
	Proxy_process_http2_session_die_other                              float64 `json:"proxy.process.http2.session_die_other,string"`
	Proxy_process_http2_session_die_eos                                float64 `json:"proxy.process.http2.session_die_eos,string"`
	Proxy_process_http2_session_die_active                             float64 `json:"proxy.process.http2.session_die_active,string"`
	Proxy_process_http2_session_die_inactive                           float64 `json:"proxy.process.http2.session_die_inactive,string"`
	Proxy_process_http2_session_die_error                              float64 `json:"proxy.process.http2.session_die_error,string"`
	Proxy_process_http2_session_die_high_error_rate                    float64 `json:"proxy.process.http2.session_die_high_error_rate,string"`
	Proxy_process_http2_max_settings_per_frame_exceeded                float64 `json:"proxy.process.http2.max_settings_per_frame_exceeded,string"`
	Proxy_process_http2_max_settings_per_minute_exceeded               float64 `json:"proxy.process.http2.max_settings_per_minute_exceeded,string"`
	Proxy_process_http2_max_settings_frames_per_minute_exceeded        float64 `json:"proxy.process.http2.max_settings_frames_per_minute_exceeded,string"`
	Proxy_process_http2_max_ping_frames_per_minute_exceeded            float64 `json:"proxy.process.http2.max_ping_frames_per_minute_exceeded,string"`
	Proxy_process_http2_max_priority_frames_per_minute_exceeded        float64 `json:"proxy.process.http2.max_priority_frames_per_minute_exceeded,string"`
	Proxy_process_http2_insufficient_avg_window_update                 float64 `json:"proxy.process.http2.insufficient_avg_window_update,string"`
	Proxy_process_log_event_log_error_ok                               float64 `json:"proxy.process.log.event_log_error_ok,string"`
	Proxy_process_log_event_log_error_skip                             float64 `json:"proxy.process.log.event_log_error_skip,string"`
	Proxy_process_log_event_log_error_aggr                             float64 `json:"proxy.process.log.event_log_error_aggr,string"`
	Proxy_process_log_event_log_error_full                             float64 `json:"proxy.process.log.event_log_error_full,string"`
	Proxy_process_log_event_log_error_fail                             float64 `json:"proxy.process.log.event_log_error_fail,string"`
	Proxy_process_log_event_log_access_ok                              float64 `json:"proxy.process.log.event_log_access_ok,string"`
	Proxy_process_log_event_log_access_skip                            float64 `json:"proxy.process.log.event_log_access_skip,string"`
	Proxy_process_log_event_log_access_aggr                            float64 `json:"proxy.process.log.event_log_access_aggr,string"`
	Proxy_process_log_event_log_access_full                            float64 `json:"proxy.process.log.event_log_access_full,string"`
	Proxy_process_log_event_log_access_fail                            float64 `json:"proxy.process.log.event_log_access_fail,string"`
	Proxy_process_log_num_sent_to_network                              float64 `json:"proxy.process.log.num_sent_to_network,string"`
	Proxy_process_log_num_lost_before_sent_to_network                  float64 `json:"proxy.process.log.num_lost_before_sent_to_network,string"`
	Proxy_process_log_num_received_from_network                        float64 `json:"proxy.process.log.num_received_from_network,string"`
	Proxy_process_log_num_flush_to_disk                                float64 `json:"proxy.process.log.num_flush_to_disk,string"`
	Proxy_process_log_num_lost_before_flush_to_disk                    float64 `json:"proxy.process.log.num_lost_before_flush_to_disk,string"`
	Proxy_process_log_bytes_lost_before_preproc                        float64 `json:"proxy.process.log.bytes_lost_before_preproc,string"`
	Proxy_process_log_bytes_sent_to_network                            float64 `json:"proxy.process.log.bytes_sent_to_network,string"`
	Proxy_process_log_bytes_lost_before_sent_to_network                float64 `json:"proxy.process.log.bytes_lost_before_sent_to_network,string"`
	Proxy_process_log_bytes_received_from_network                      float64 `json:"proxy.process.log.bytes_received_from_network,string"`
	Proxy_process_log_bytes_flush_to_disk                              float64 `json:"proxy.process.log.bytes_flush_to_disk,string"`
	Proxy_process_log_bytes_lost_before_flush_to_disk                  float64 `json:"proxy.process.log.bytes_lost_before_flush_to_disk,string"`
	Proxy_process_log_bytes_written_to_disk                            float64 `json:"proxy.process.log.bytes_written_to_disk,string"`
	Proxy_process_log_bytes_lost_before_written_to_disk                float64 `json:"proxy.process.log.bytes_lost_before_written_to_disk,string"`
	Proxy_process_ssl_user_agent_other_errors                          float64 `json:"proxy.process.ssl.user_agent_other_errors,string"`
	Proxy_process_ssl_user_agent_expired_cert                          float64 `json:"proxy.process.ssl.user_agent_expired_cert,string"`
	Proxy_process_ssl_user_agent_revoked_cert                          float64 `json:"proxy.process.ssl.user_agent_revoked_cert,string"`
	Proxy_process_ssl_user_agent_unknown_cert                          float64 `json:"proxy.process.ssl.user_agent_unknown_cert,string"`
	Proxy_process_ssl_user_agent_cert_verify_failed                    float64 `json:"proxy.process.ssl.user_agent_cert_verify_failed,string"`
	Proxy_process_ssl_user_agent_bad_cert                              float64 `json:"proxy.process.ssl.user_agent_bad_cert,string"`
	Proxy_process_ssl_user_agent_decryption_failed                     float64 `json:"proxy.process.ssl.user_agent_decryption_failed,string"`
	Proxy_process_ssl_user_agent_wrong_version                         float64 `json:"proxy.process.ssl.user_agent_wrong_version,string"`
	Proxy_process_ssl_user_agent_unknown_ca                            float64 `json:"proxy.process.ssl.user_agent_unknown_ca,string"`
	Proxy_process_ssl_origin_server_other_errors                       float64 `json:"proxy.process.ssl.origin_server_other_errors,string"`
	Proxy_process_ssl_origin_server_expired_cert                       float64 `json:"proxy.process.ssl.origin_server_expired_cert,string"`
	Proxy_process_ssl_origin_server_revoked_cert                       float64 `json:"proxy.process.ssl.origin_server_revoked_cert,string"`
	Proxy_process_ssl_origin_server_unknown_cert                       float64 `json:"proxy.process.ssl.origin_server_unknown_cert,string"`
	Proxy_process_ssl_origin_server_cert_verify_failed                 float64 `json:"proxy.process.ssl.origin_server_cert_verify_failed,string"`
	Proxy_process_ssl_origin_server_bad_cert                           float64 `json:"proxy.process.ssl.origin_server_bad_cert,string"`
	Proxy_process_ssl_origin_server_decryption_failed                  float64 `json:"proxy.process.ssl.origin_server_decryption_failed,string"`
	Proxy_process_ssl_origin_server_wrong_version                      float64 `json:"proxy.process.ssl.origin_server_wrong_version,string"`
	Proxy_process_ssl_origin_server_unknown_ca                         float64 `json:"proxy.process.ssl.origin_server_unknown_ca,string"`
	Proxy_process_ssl_total_handshake_time                             float64 `json:"proxy.process.ssl.total_handshake_time,string"`
	Proxy_process_ssl_total_attempts_handshake_count_in                float64 `json:"proxy.process.ssl.total_attempts_handshake_count_in,string"`
	Proxy_process_ssl_total_success_handshake_count_in                 float64 `json:"proxy.process.ssl.total_success_handshake_count_in,string"`
	Proxy_process_ssl_total_attempts_handshake_count_out               float64 `json:"proxy.process.ssl.total_attempts_handshake_count_out,string"`
	Proxy_process_ssl_total_success_handshake_count_out                float64 `json:"proxy.process.ssl.total_success_handshake_count_out,string"`
	Proxy_process_ssl_total_tickets_created                            float64 `json:"proxy.process.ssl.total_tickets_created,string"`
	Proxy_process_ssl_total_tickets_verified                           float64 `json:"proxy.process.ssl.total_tickets_verified,string"`
	Proxy_process_ssl_total_tickets_not_found                          float64 `json:"proxy.process.ssl.total_tickets_not_found,string"`
	Proxy_process_ssl_total_tickets_renewed                            float64 `json:"proxy.process.ssl.total_tickets_renewed,string"`
	Proxy_process_ssl_total_tickets_verified_old_key                   float64 `json:"proxy.process.ssl.total_tickets_verified_old_key,string"`
	Proxy_process_ssl_total_ticket_keys_renewed                        float64 `json:"proxy.process.ssl.total_ticket_keys_renewed,string"`
	Proxy_process_ssl_ssl_session_cache_hit                            float64 `json:"proxy.process.ssl.ssl_session_cache_hit,string"`
	Proxy_process_ssl_ssl_session_cache_new_session                    float64 `json:"proxy.process.ssl.ssl_session_cache_new_session,string"`
	Proxy_process_ssl_ssl_session_cache_miss                           float64 `json:"proxy.process.ssl.ssl_session_cache_miss,string"`
	Proxy_process_ssl_ssl_session_cache_eviction                       float64 `json:"proxy.process.ssl.ssl_session_cache_eviction,string"`
	Proxy_process_ssl_ssl_session_cache_lock_contention                float64 `json:"proxy.process.ssl.ssl_session_cache_lock_contention,string"`
	Proxy_process_ssl_default_record_size_count                        float64 `json:"proxy.process.ssl.default_record_size_count,string"`
	Proxy_process_ssl_max_record_size_count                            float64 `json:"proxy.process.ssl.max_record_size_count,string"`
	Proxy_process_ssl_redo_record_size_count                           float64 `json:"proxy.process.ssl.redo_record_size_count,string"`
	Proxy_process_ssl_ssl_error_syscall                                float64 `json:"proxy.process.ssl.ssl_error_syscall,string"`
	Proxy_process_ssl_ssl_error_ssl                                    float64 `json:"proxy.process.ssl.ssl_error_ssl,string"`
	Proxy_process_ssl_ssl_sni_name_set_failure                         float64 `json:"proxy.process.ssl.ssl_sni_name_set_failure,string"`
	Proxy_process_ssl_ssl_ocsp_revoked_cert_stat                       float64 `json:"proxy.process.ssl.ssl_ocsp_revoked_cert_stat,string"`
	Proxy_process_ssl_ssl_ocsp_unknown_cert_stat                       float64 `json:"proxy.process.ssl.ssl_ocsp_unknown_cert_stat,string"`
	Proxy_process_ssl_ssl_ocsp_refreshed_cert                          float64 `json:"proxy.process.ssl.ssl_ocsp_refreshed_cert,string"`
	Proxy_process_ssl_ssl_ocsp_refresh_cert_failure                    float64 `json:"proxy.process.ssl.ssl_ocsp_refresh_cert_failure,string"`
	Proxy_process_ssl_ssl_total_sslv3                                  float64 `json:"proxy.process.ssl.ssl_total_sslv3,string"`
	Proxy_process_ssl_ssl_total_tlsv1                                  float64 `json:"proxy.process.ssl.ssl_total_tlsv1,string"`
	Proxy_process_ssl_ssl_total_tlsv11                                 float64 `json:"proxy.process.ssl.ssl_total_tlsv11,string"`
	Proxy_process_ssl_ssl_total_tlsv12                                 float64 `json:"proxy.process.ssl.ssl_total_tlsv12,string"`
	Proxy_process_ssl_ssl_total_tlsv13                                 float64 `json:"proxy.process.ssl.ssl_total_tlsv13,string"`
	Proxy_process_ssl_early_data_received                              float64 `json:"proxy.process.ssl.early_data_received,string"`
	Proxy_node_config_reconfigure_required                             float64 `json:"proxy.node.config.reconfigure_required,string"`
	Proxy_node_config_restart_required_proxy                           float64 `json:"proxy.node.config.restart_required.proxy,string"`
	Proxy_node_config_restart_required_manager                         float64 `json:"proxy.node.config.restart_required.manager,string"`
	Proxy_node_config_draining                                         float64 `json:"proxy.node.config.draining,string"`
	Proxy_process_http_background_fill_current_count                   float64 `json:"proxy.process.http.background_fill_current_count,string"`
	Proxy_process_http_current_client_connections                      float64 `json:"proxy.process.http.current_client_connections,string"`
	Proxy_process_http_current_active_client_connections               float64 `json:"proxy.process.http.current_active_client_connections,string"`
	Proxy_process_http_websocket_current_active_client_connections     float64 `json:"proxy.process.http.websocket.current_active_client_connections,string"`
	Proxy_process_http_current_client_transactions                     float64 `json:"proxy.process.http.current_client_transactions,string"`
	Proxy_process_http_current_server_transactions                     float64 `json:"proxy.process.http.current_server_transactions,string"`
	Proxy_process_http_origin_shutdown_pool_lock_contention            float64 `json:"proxy.process.http.origin_shutdown.pool_lock_contention,string"`
	Proxy_process_http_origin_shutdown_migration_failure               float64 `json:"proxy.process.http.origin_shutdown.migration_failure,string"`
	Proxy_process_http_origin_shutdown_tunnel_server                   float64 `json:"proxy.process.http.origin_shutdown.tunnel_server,string"`
	Proxy_process_http_origin_shutdown_tunnel_server_no_keep_alive     float64 `json:"proxy.process.http.origin_shutdown.tunnel_server_no_keep_alive,string"`
	Proxy_process_http_origin_shutdown_tunnel_server_eos               float64 `json:"proxy.process.http.origin_shutdown.tunnel_server_eos,string"`
	Proxy_process_http_origin_shutdown_tunnel_server_plugin_tunnel     float64 `json:"proxy.process.http.origin_shutdown.tunnel_server_plugin_tunnel,string"`
	Proxy_process_http_origin_shutdown_tunnel_server_detach            float64 `json:"proxy.process.http.origin_shutdown.tunnel_server_detach,string"`
	Proxy_process_http_origin_shutdown_tunnel_client                   float64 `json:"proxy.process.http.origin_shutdown.tunnel_client,string"`
	Proxy_process_http_origin_shutdown_tunnel_transform_read           float64 `json:"proxy.process.http.origin_shutdown.tunnel_transform_read,string"`
	Proxy_process_http_origin_shutdown_release_no_sharing              float64 `json:"proxy.process.http.origin_shutdown.release_no_sharing,string"`
	Proxy_process_http_origin_shutdown_release_no_server               float64 `json:"proxy.process.http.origin_shutdown.release_no_server,string"`
	Proxy_process_http_origin_shutdown_release_no_keep_alive           float64 `json:"proxy.process.http.origin_shutdown.release_no_keep_alive,string"`
	Proxy_process_http_origin_shutdown_release_invalid_response        float64 `json:"proxy.process.http.origin_shutdown.release_invalid_response,string"`
	Proxy_process_http_origin_shutdown_release_invalid_request         float64 `json:"proxy.process.http.origin_shutdown.release_invalid_request,string"`
	Proxy_process_http_origin_shutdown_release_modified                float64 `json:"proxy.process.http.origin_shutdown.release_modified,string"`
	Proxy_process_http_origin_shutdown_release_misc                    float64 `json:"proxy.process.http.origin_shutdown.release_misc,string"`
	Proxy_process_http_origin_shutdown_cleanup_entry                   float64 `json:"proxy.process.http.origin_shutdown.cleanup_entry,string"`
	Proxy_process_http_origin_shutdown_tunnel_abort                    float64 `json:"proxy.process.http.origin_shutdown.tunnel_abort,string"`
	Proxy_process_http_current_parent_proxy_connections                float64 `json:"proxy.process.http.current_parent_proxy_connections,string"`
	Proxy_process_http_current_server_connections                      float64 `json:"proxy.process.http.current_server_connections,string"`
	Proxy_process_http_current_cache_connections                       float64 `json:"proxy.process.http.current_cache_connections,string"`
	Proxy_process_http_origin_connect_adjust_thread                    float64 `json:"proxy.process.http.origin.connect.adjust_thread,string"`
	Proxy_process_http_cache_open_write_adjust_thread                  float64 `json:"proxy.process.http.cache.open_write.adjust_thread,string"`
	Proxy_process_net_accepts_currently_open                           float64 `json:"proxy.process.net.accepts_currently_open,string"`
	Proxy_process_net_connections_currently_open                       float64 `json:"proxy.process.net.connections_currently_open,string"`
	Proxy_process_net_default_inactivity_timeout_applied               float64 `json:"proxy.process.net.default_inactivity_timeout_applied,string"`
	Proxy_process_net_default_inactivity_timeout_count                 float64 `json:"proxy.process.net.default_inactivity_timeout_count,string"`
	Proxy_process_net_dynamic_keep_alive_timeout_in_count              float64 `json:"proxy.process.net.dynamic_keep_alive_timeout_in_count,string"`
	Proxy_process_net_dynamic_keep_alive_timeout_in_total              float64 `json:"proxy.process.net.dynamic_keep_alive_timeout_in_total,string"`
	Proxy_process_socks_connections_currently_open                     float64 `json:"proxy.process.socks.connections_currently_open,string"`
	Proxy_process_tcp_total_accepts                                    float64 `json:"proxy.process.tcp.total_accepts,string"`
	Proxy_process_cache_bytes_used                                     float64 `json:"proxy.process.cache.bytes_used,string"`
	Proxy_process_cache_bytes_total                                    float64 `json:"proxy.process.cache.bytes_total,string"`
	Proxy_process_cache_ram_cache_total_bytes                          float64 `json:"proxy.process.cache.ram_cache.total_bytes,string"`
	Proxy_process_cache_ram_cache_bytes_used                           float64 `json:"proxy.process.cache.ram_cache.bytes_used,string"`
	Proxy_process_cache_ram_cache_hits                                 float64 `json:"proxy.process.cache.ram_cache.hits,string"`
	Proxy_process_cache_ram_cache_misses                               float64 `json:"proxy.process.cache.ram_cache.misses,string"`
	Proxy_process_cache_pread_count                                    float64 `json:"proxy.process.cache.pread_count,string"`
	Proxy_process_cache_percent_full                                   float64 `json:"proxy.process.cache.percent_full,string"`
	Proxy_process_cache_lookup_active                                  float64 `json:"proxy.process.cache.lookup.active,string"`
	Proxy_process_cache_lookup_success                                 float64 `json:"proxy.process.cache.lookup.success,string"`
	Proxy_process_cache_lookup_failure                                 float64 `json:"proxy.process.cache.lookup.failure,string"`
	Proxy_process_cache_read_active                                    float64 `json:"proxy.process.cache.read.active,string"`
	Proxy_process_cache_read_success                                   float64 `json:"proxy.process.cache.read.success,string"`
	Proxy_process_cache_read_failure                                   float64 `json:"proxy.process.cache.read.failure,string"`
	Proxy_process_cache_write_active                                   float64 `json:"proxy.process.cache.write.active,string"`
	Proxy_process_cache_write_success                                  float64 `json:"proxy.process.cache.write.success,string"`
	Proxy_process_cache_write_failure                                  float64 `json:"proxy.process.cache.write.failure,string"`
	Proxy_process_cache_write_backlog_failure                          float64 `json:"proxy.process.cache.write.backlog.failure,string"`
	Proxy_process_cache_update_active                                  float64 `json:"proxy.process.cache.update.active,string"`
	Proxy_process_cache_update_success                                 float64 `json:"proxy.process.cache.update.success,string"`
	Proxy_process_cache_update_failure                                 float64 `json:"proxy.process.cache.update.failure,string"`
	Proxy_process_cache_remove_active                                  float64 `json:"proxy.process.cache.remove.active,string"`
	Proxy_process_cache_remove_success                                 float64 `json:"proxy.process.cache.remove.success,string"`
	Proxy_process_cache_remove_failure                                 float64 `json:"proxy.process.cache.remove.failure,string"`
	Proxy_process_cache_evacuate_active                                float64 `json:"proxy.process.cache.evacuate.active,string"`
	Proxy_process_cache_evacuate_success                               float64 `json:"proxy.process.cache.evacuate.success,string"`
	Proxy_process_cache_evacuate_failure                               float64 `json:"proxy.process.cache.evacuate.failure,string"`
	Proxy_process_cache_scan_active                                    float64 `json:"proxy.process.cache.scan.active,string"`
	Proxy_process_cache_scan_success                                   float64 `json:"proxy.process.cache.scan.success,string"`
	Proxy_process_cache_scan_failure                                   float64 `json:"proxy.process.cache.scan.failure,string"`
	Proxy_process_cache_direntries_total                               float64 `json:"proxy.process.cache.direntries.total,string"`
	Proxy_process_cache_direntries_used                                float64 `json:"proxy.process.cache.direntries.used,string"`
	Proxy_process_cache_directory_collision                            float64 `json:"proxy.process.cache.directory_collision,string"`
	Proxy_process_cache_frags_per_doc_1                                float64 `json:"proxy.process.cache.frags_per_doc.1,string"`
	Proxy_process_cache_frags_per_doc_2                                float64 `json:"proxy.process.cache.frags_per_doc.2,string"`
	Proxy_process_cache_read_busy_success                              float64 `json:"proxy.process.cache.read_busy.success,string"`
	Proxy_process_cache_read_busy_failure                              float64 `json:"proxy.process.cache.read_busy.failure,string"`
	Proxy_process_cache_write_bytes_stat                               float64 `json:"proxy.process.cache.write_bytes_stat,string"`
	Proxy_process_cache_vector_marshals                                float64 `json:"proxy.process.cache.vector_marshals,string"`
	Proxy_process_cache_hdr_marshals                                   float64 `json:"proxy.process.cache.hdr_marshals,string"`
	Proxy_process_cache_hdr_marshal_bytes                              float64 `json:"proxy.process.cache.hdr_marshal_bytes,string"`
	Proxy_process_cache_gc_bytes_evacuated                             float64 `json:"proxy.process.cache.gc_bytes_evacuated,string"`
	Proxy_process_cache_gc_frags_evacuated                             float64 `json:"proxy.process.cache.gc_frags_evacuated,string"`
	Proxy_process_cache_wrap_count                                     float64 `json:"proxy.process.cache.wrap_count,string"`
	Proxy_process_cache_sync_count                                     float64 `json:"proxy.process.cache.sync.count,string"`
	Proxy_process_cache_sync_bytes                                     float64 `json:"proxy.process.cache.sync.bytes,string"`
	Proxy_process_cache_sync_time                                      float64 `json:"proxy.process.cache.sync.time,string"`
	Proxy_process_cache_span_errors_read                               float64 `json:"proxy.process.cache.span.errors.read,string"`
	Proxy_process_cache_span_errors_write                              float64 `json:"proxy.process.cache.span.errors.write,string"`
	Proxy_process_cache_span_failing                                   float64 `json:"proxy.process.cache.span.failing,string"`
	Proxy_process_cache_span_offline                                   float64 `json:"proxy.process.cache.span.offline,string"`
	Proxy_process_cache_span_online                                    float64 `json:"proxy.process.cache.span.online,string"`
	Proxy_process_dns_success_avg_time                                 float64 `json:"proxy.process.dns.success_avg_time,string"`
	Proxy_process_dns_in_flight                                        float64 `json:"proxy.process.dns.in_flight,string"`
	Proxy_process_eventloop_count_10s                                  float64 `json:"proxy.process.eventloop.count.10s,string"`
	Proxy_process_eventloop_events_10s                                 float64 `json:"proxy.process.eventloop.events.10s,string"`
	Proxy_process_eventloop_events_min_10s                             float64 `json:"proxy.process.eventloop.events.min.10s,string"`
	Proxy_process_eventloop_events_max_10s                             float64 `json:"proxy.process.eventloop.events.max.10s,string"`
	Proxy_process_eventloop_wait_10s                                   float64 `json:"proxy.process.eventloop.wait.10s,string"`
	Proxy_process_eventloop_time_min_10s                               float64 `json:"proxy.process.eventloop.time.min.10s,string"`
	Proxy_process_eventloop_time_max_10s                               float64 `json:"proxy.process.eventloop.time.max.10s,string"`
	Proxy_process_eventloop_count_100s                                 float64 `json:"proxy.process.eventloop.count.100s,string"`
	Proxy_process_eventloop_events_100s                                float64 `json:"proxy.process.eventloop.events.100s,string"`
	Proxy_process_eventloop_events_min_100s                            float64 `json:"proxy.process.eventloop.events.min.100s,string"`
	Proxy_process_eventloop_events_max_100s                            float64 `json:"proxy.process.eventloop.events.max.100s,string"`
	Proxy_process_eventloop_wait_100s                                  float64 `json:"proxy.process.eventloop.wait.100s,string"`
	Proxy_process_eventloop_time_min_100s                              float64 `json:"proxy.process.eventloop.time.min.100s,string"`
	Proxy_process_eventloop_time_max_100s                              float64 `json:"proxy.process.eventloop.time.max.100s,string"`
	Proxy_process_eventloop_count_1000s                                float64 `json:"proxy.process.eventloop.count.1000s,string"`
	Proxy_process_eventloop_events_1000s                               float64 `json:"proxy.process.eventloop.events.1000s,string"`
	Proxy_process_eventloop_events_min_1000s                           float64 `json:"proxy.process.eventloop.events.min.1000s,string"`
	Proxy_process_eventloop_events_max_1000s                           float64 `json:"proxy.process.eventloop.events.max.1000s,string"`
	Proxy_process_eventloop_wait_1000s                                 float64 `json:"proxy.process.eventloop.wait.1000s,string"`
	Proxy_process_eventloop_time_min_1000s                             float64 `json:"proxy.process.eventloop.time.min.1000s,string"`
	Proxy_process_eventloop_time_max_1000s                             float64 `json:"proxy.process.eventloop.time.max.1000s,string"`
	Proxy_process_traffic_server_memory_rss                            float64 `json:"proxy.process.traffic_server.memory.rss,string"`
	Proxy_process_http2_current_client_connections                     float64 `json:"proxy.process.http2.current_client_connections,string"`
	Proxy_process_http2_current_active_client_connections              float64 `json:"proxy.process.http2.current_active_client_connections,string"`
	Proxy_process_http2_current_client_streams                         float64 `json:"proxy.process.http2.current_client_streams,string"`
	Proxy_process_hostdb_cache_current_items                           float64 `json:"proxy.process.hostdb.cache.current_items,string"`
	Proxy_process_hostdb_cache_current_size                            float64 `json:"proxy.process.hostdb.cache.current_size,string"`
	Proxy_process_hostdb_cache_total_inserts                           float64 `json:"proxy.process.hostdb.cache.total_inserts,string"`
	Proxy_process_hostdb_cache_total_failed_inserts                    float64 `json:"proxy.process.hostdb.cache.total_failed_inserts,string"`
	Proxy_process_hostdb_cache_total_lookups                           float64 `json:"proxy.process.hostdb.cache.total_lookups,string"`
	Proxy_process_hostdb_cache_total_hits                              float64 `json:"proxy.process.hostdb.cache.total_hits,string"`
	Proxy_process_hostdb_cache_last_sync_time                          float64 `json:"proxy.process.hostdb.cache.last_sync.time,string"`
	Proxy_process_hostdb_cache_last_sync_total_items                   float64 `json:"proxy.process.hostdb.cache.last_sync.total_items,string"`
	Proxy_process_hostdb_cache_last_sync_total_size                    float64 `json:"proxy.process.hostdb.cache.last_sync.total_size,string"`
	Proxy_process_log_log_files_open                                   float64 `json:"proxy.process.log.log_files_open,string"`
	Proxy_process_log_log_files_space_used                             float64 `json:"proxy.process.log.log_files_space_used,string"`
	Plugin_lua_global_states                                           float64 `json:"plugin.lua.global.states,string"`
	Plugin_lua_global_gc_bytes                                         float64 `json:"plugin.lua.global.gc_bytes,string"`
	Plugin_lua_global_threads                                          float64 `json:"plugin.lua.global.threads,string"`
	Proxy_process_ssl_user_agent_sessions                              float64 `json:"proxy.process.ssl.user_agent_sessions,string"`
	Proxy_process_ssl_user_agent_session_hit                           float64 `json:"proxy.process.ssl.user_agent_session_hit,string"`
	Proxy_process_ssl_user_agent_session_miss                          float64 `json:"proxy.process.ssl.user_agent_session_miss,string"`
	Proxy_process_ssl_user_agent_session_timeout                       float64 `json:"proxy.process.ssl.user_agent_session_timeout,string"`
	Proxy_process_ssl_cipher_user_agent_TLS_AES_256_GCM_SHA384         float64 `json:"proxy.process.ssl.cipher.user_agent.TLS_AES_256_GCM_SHA384,string"`
	Proxy_process_ssl_cipher_user_agent_TLS_CHACHA20_POLY1305_SHA256   float64 `json:"proxy.process.ssl.cipher.user_agent.TLS_CHACHA20_POLY1305_SHA256,string"`
	Proxy_process_ssl_cipher_user_agent_TLS_AES_128_GCM_SHA256         float64 `json:"proxy.process.ssl.cipher.user_agent.TLS_AES_128_GCM_SHA256,string"`
	Proxy_process_ssl_cipher_user_agent_ECDHE_ECDSA_AES256_GCM_SHA384  float64 `json:"proxy.process.ssl.cipher.user_agent.ECDHE-ECDSA-AES256-GCM-SHA384,string"`
	Proxy_process_ssl_cipher_user_agent_ECDHE_RSA_AES256_GCM_SHA384    float64 `json:"proxy.process.ssl.cipher.user_agent.ECDHE-RSA-AES256-GCM-SHA384,string"`
	Proxy_process_ssl_cipher_user_agent_DHE_RSA_AES256_GCM_SHA384      float64 `json:"proxy.process.ssl.cipher.user_agent.DHE-RSA-AES256-GCM-SHA384,string"`
	Proxy_process_ssl_cipher_user_agent_ECDHE_ECDSA_CHACHA20_POLY1305  float64 `json:"proxy.process.ssl.cipher.user_agent.ECDHE-ECDSA-CHACHA20-POLY1305,string"`
	Proxy_process_ssl_cipher_user_agent_ECDHE_RSA_CHACHA20_POLY1305    float64 `json:"proxy.process.ssl.cipher.user_agent.ECDHE-RSA-CHACHA20-POLY1305,string"`
	Proxy_process_ssl_cipher_user_agent_DHE_RSA_CHACHA20_POLY1305      float64 `json:"proxy.process.ssl.cipher.user_agent.DHE-RSA-CHACHA20-POLY1305,string"`
	Proxy_process_ssl_cipher_user_agent_ECDHE_ECDSA_AES128_GCM_SHA256  float64 `json:"proxy.process.ssl.cipher.user_agent.ECDHE-ECDSA-AES128-GCM-SHA256,string"`
	Proxy_process_ssl_cipher_user_agent_ECDHE_RSA_AES128_GCM_SHA256    float64 `json:"proxy.process.ssl.cipher.user_agent.ECDHE-RSA-AES128-GCM-SHA256,string"`
	Proxy_process_ssl_cipher_user_agent_DHE_RSA_AES128_GCM_SHA256      float64 `json:"proxy.process.ssl.cipher.user_agent.DHE-RSA-AES128-GCM-SHA256,string"`
	Proxy_process_ssl_cipher_user_agent_ECDHE_ECDSA_AES256_SHA384      float64 `json:"proxy.process.ssl.cipher.user_agent.ECDHE-ECDSA-AES256-SHA384,string"`
	Proxy_process_ssl_cipher_user_agent_ECDHE_RSA_AES256_SHA384        float64 `json:"proxy.process.ssl.cipher.user_agent.ECDHE-RSA-AES256-SHA384,string"`
	Proxy_process_ssl_cipher_user_agent_DHE_RSA_AES256_SHA256          float64 `json:"proxy.process.ssl.cipher.user_agent.DHE-RSA-AES256-SHA256,string"`
	Proxy_process_ssl_cipher_user_agent_ECDHE_ECDSA_AES128_SHA256      float64 `json:"proxy.process.ssl.cipher.user_agent.ECDHE-ECDSA-AES128-SHA256,string"`
	Proxy_process_ssl_cipher_user_agent_ECDHE_RSA_AES128_SHA256        float64 `json:"proxy.process.ssl.cipher.user_agent.ECDHE-RSA-AES128-SHA256,string"`
	Proxy_process_ssl_cipher_user_agent_DHE_RSA_AES128_SHA256          float64 `json:"proxy.process.ssl.cipher.user_agent.DHE-RSA-AES128-SHA256,string"`
	Proxy_process_ssl_cipher_user_agent_ECDHE_ECDSA_AES256_SHA         float64 `json:"proxy.process.ssl.cipher.user_agent.ECDHE-ECDSA-AES256-SHA,string"`
	Proxy_process_ssl_cipher_user_agent_ECDHE_RSA_AES256_SHA           float64 `json:"proxy.process.ssl.cipher.user_agent.ECDHE-RSA-AES256-SHA,string"`
	Proxy_process_ssl_cipher_user_agent_DHE_RSA_AES256_SHA             float64 `json:"proxy.process.ssl.cipher.user_agent.DHE-RSA-AES256-SHA,string"`
	Proxy_process_ssl_cipher_user_agent_ECDHE_ECDSA_AES128_SHA         float64 `json:"proxy.process.ssl.cipher.user_agent.ECDHE-ECDSA-AES128-SHA,string"`
	Proxy_process_ssl_cipher_user_agent_ECDHE_RSA_AES128_SHA           float64 `json:"proxy.process.ssl.cipher.user_agent.ECDHE-RSA-AES128-SHA,string"`
	Proxy_process_ssl_cipher_user_agent_DHE_RSA_AES128_SHA             float64 `json:"proxy.process.ssl.cipher.user_agent.DHE-RSA-AES128-SHA,string"`
	Proxy_process_ssl_cipher_user_agent_RSA_PSK_AES256_GCM_SHA384      float64 `json:"proxy.process.ssl.cipher.user_agent.RSA-PSK-AES256-GCM-SHA384,string"`
	Proxy_process_ssl_cipher_user_agent_DHE_PSK_AES256_GCM_SHA384      float64 `json:"proxy.process.ssl.cipher.user_agent.DHE-PSK-AES256-GCM-SHA384,string"`
	Proxy_process_ssl_cipher_user_agent_RSA_PSK_CHACHA20_POLY1305      float64 `json:"proxy.process.ssl.cipher.user_agent.RSA-PSK-CHACHA20-POLY1305,string"`
	Proxy_process_ssl_cipher_user_agent_DHE_PSK_CHACHA20_POLY1305      float64 `json:"proxy.process.ssl.cipher.user_agent.DHE-PSK-CHACHA20-POLY1305,string"`
	Proxy_process_ssl_cipher_user_agent_ECDHE_PSK_CHACHA20_POLY1305    float64 `json:"proxy.process.ssl.cipher.user_agent.ECDHE-PSK-CHACHA20-POLY1305,string"`
	Proxy_process_ssl_cipher_user_agent_AES256_GCM_SHA384              float64 `json:"proxy.process.ssl.cipher.user_agent.AES256-GCM-SHA384,string"`
	Proxy_process_ssl_cipher_user_agent_PSK_AES256_GCM_SHA384          float64 `json:"proxy.process.ssl.cipher.user_agent.PSK-AES256-GCM-SHA384,string"`
	Proxy_process_ssl_cipher_user_agent_PSK_CHACHA20_POLY1305          float64 `json:"proxy.process.ssl.cipher.user_agent.PSK-CHACHA20-POLY1305,string"`
	Proxy_process_ssl_cipher_user_agent_RSA_PSK_AES128_GCM_SHA256      float64 `json:"proxy.process.ssl.cipher.user_agent.RSA-PSK-AES128-GCM-SHA256,string"`
	Proxy_process_ssl_cipher_user_agent_DHE_PSK_AES128_GCM_SHA256      float64 `json:"proxy.process.ssl.cipher.user_agent.DHE-PSK-AES128-GCM-SHA256,string"`
	Proxy_process_ssl_cipher_user_agent_AES128_GCM_SHA256              float64 `json:"proxy.process.ssl.cipher.user_agent.AES128-GCM-SHA256,string"`
	Proxy_process_ssl_cipher_user_agent_PSK_AES128_GCM_SHA256          float64 `json:"proxy.process.ssl.cipher.user_agent.PSK-AES128-GCM-SHA256,string"`
	Proxy_process_ssl_cipher_user_agent_AES256_SHA256                  float64 `json:"proxy.process.ssl.cipher.user_agent.AES256-SHA256,string"`
	Proxy_process_ssl_cipher_user_agent_AES128_SHA256                  float64 `json:"proxy.process.ssl.cipher.user_agent.AES128-SHA256,string"`
	Proxy_process_ssl_cipher_user_agent_ECDHE_PSK_AES256_CBC_SHA384    float64 `json:"proxy.process.ssl.cipher.user_agent.ECDHE-PSK-AES256-CBC-SHA384,string"`
	Proxy_process_ssl_cipher_user_agent_ECDHE_PSK_AES256_CBC_SHA       float64 `json:"proxy.process.ssl.cipher.user_agent.ECDHE-PSK-AES256-CBC-SHA,string"`
	Proxy_process_ssl_cipher_user_agent_SRP_RSA_AES_256_CBC_SHA        float64 `json:"proxy.process.ssl.cipher.user_agent.SRP-RSA-AES-256-CBC-SHA,string"`
	Proxy_process_ssl_cipher_user_agent_SRP_AES_256_CBC_SHA            float64 `json:"proxy.process.ssl.cipher.user_agent.SRP-AES-256-CBC-SHA,string"`
	Proxy_process_ssl_cipher_user_agent_RSA_PSK_AES256_CBC_SHA384      float64 `json:"proxy.process.ssl.cipher.user_agent.RSA-PSK-AES256-CBC-SHA384,string"`
	Proxy_process_ssl_cipher_user_agent_DHE_PSK_AES256_CBC_SHA384      float64 `json:"proxy.process.ssl.cipher.user_agent.DHE-PSK-AES256-CBC-SHA384,string"`
	Proxy_process_ssl_cipher_user_agent_RSA_PSK_AES256_CBC_SHA         float64 `json:"proxy.process.ssl.cipher.user_agent.RSA-PSK-AES256-CBC-SHA,string"`
	Proxy_process_ssl_cipher_user_agent_DHE_PSK_AES256_CBC_SHA         float64 `json:"proxy.process.ssl.cipher.user_agent.DHE-PSK-AES256-CBC-SHA,string"`
	Proxy_process_ssl_cipher_user_agent_AES256_SHA                     float64 `json:"proxy.process.ssl.cipher.user_agent.AES256-SHA,string"`
	Proxy_process_ssl_cipher_user_agent_PSK_AES256_CBC_SHA384          float64 `json:"proxy.process.ssl.cipher.user_agent.PSK-AES256-CBC-SHA384,string"`
	Proxy_process_ssl_cipher_user_agent_PSK_AES256_CBC_SHA             float64 `json:"proxy.process.ssl.cipher.user_agent.PSK-AES256-CBC-SHA,string"`
	Proxy_process_ssl_cipher_user_agent_ECDHE_PSK_AES128_CBC_SHA256    float64 `json:"proxy.process.ssl.cipher.user_agent.ECDHE-PSK-AES128-CBC-SHA256,string"`
	Proxy_process_ssl_cipher_user_agent_ECDHE_PSK_AES128_CBC_SHA       float64 `json:"proxy.process.ssl.cipher.user_agent.ECDHE-PSK-AES128-CBC-SHA,string"`
	Proxy_process_ssl_cipher_user_agent_SRP_RSA_AES_128_CBC_SHA        float64 `json:"proxy.process.ssl.cipher.user_agent.SRP-RSA-AES-128-CBC-SHA,string"`
	Proxy_process_ssl_cipher_user_agent_SRP_AES_128_CBC_SHA            float64 `json:"proxy.process.ssl.cipher.user_agent.SRP-AES-128-CBC-SHA,string"`
	Proxy_process_ssl_cipher_user_agent_RSA_PSK_AES128_CBC_SHA256      float64 `json:"proxy.process.ssl.cipher.user_agent.RSA-PSK-AES128-CBC-SHA256,string"`
	Proxy_process_ssl_cipher_user_agent_DHE_PSK_AES128_CBC_SHA256      float64 `json:"proxy.process.ssl.cipher.user_agent.DHE-PSK-AES128-CBC-SHA256,string"`
	Proxy_process_ssl_cipher_user_agent_RSA_PSK_AES128_CBC_SHA         float64 `json:"proxy.process.ssl.cipher.user_agent.RSA-PSK-AES128-CBC-SHA,string"`
	Proxy_process_ssl_cipher_user_agent_DHE_PSK_AES128_CBC_SHA         float64 `json:"proxy.process.ssl.cipher.user_agent.DHE-PSK-AES128-CBC-SHA,string"`
	Proxy_process_ssl_cipher_user_agent_AES128_SHA                     float64 `json:"proxy.process.ssl.cipher.user_agent.AES128-SHA,string"`
	Proxy_process_ssl_cipher_user_agent_PSK_AES128_CBC_SHA256          float64 `json:"proxy.process.ssl.cipher.user_agent.PSK-AES128-CBC-SHA256,string"`
	Proxy_process_ssl_cipher_user_agent_PSK_AES128_CBC_SHA             float64 `json:"proxy.process.ssl.cipher.user_agent.PSK-AES128-CBC-SHA,string"`
	Proxy_process_cache_volume_1_bytes_used                            float64 `json:"proxy.process.cache.volume_1.bytes_used,string"`
	Proxy_process_cache_volume_1_bytes_total                           float64 `json:"proxy.process.cache.volume_1.bytes_total,string"`
	Proxy_process_cache_volume_1_ram_cache_total_bytes                 float64 `json:"proxy.process.cache.volume_1.ram_cache.total_bytes,string"`
	Proxy_process_cache_volume_1_ram_cache_bytes_used                  float64 `json:"proxy.process.cache.volume_1.ram_cache.bytes_used,string"`
	Proxy_process_cache_volume_1_ram_cache_hits                        float64 `json:"proxy.process.cache.volume_1.ram_cache.hits,string"`
	Proxy_process_cache_volume_1_ram_cache_misses                      float64 `json:"proxy.process.cache.volume_1.ram_cache.misses,string"`
	Proxy_process_cache_volume_1_pread_count                           float64 `json:"proxy.process.cache.volume_1.pread_count,string"`
	Proxy_process_cache_volume_1_percent_full                          float64 `json:"proxy.process.cache.volume_1.percent_full,string"`
	Proxy_process_cache_volume_1_lookup_active                         float64 `json:"proxy.process.cache.volume_1.lookup.active,string"`
	Proxy_process_cache_volume_1_lookup_success                        float64 `json:"proxy.process.cache.volume_1.lookup.success,string"`
	Proxy_process_cache_volume_1_lookup_failure                        float64 `json:"proxy.process.cache.volume_1.lookup.failure,string"`
	Proxy_process_cache_volume_1_read_active                           float64 `json:"proxy.process.cache.volume_1.read.active,string"`
	Proxy_process_cache_volume_1_read_success                          float64 `json:"proxy.process.cache.volume_1.read.success,string"`
	Proxy_process_cache_volume_1_read_failure                          float64 `json:"proxy.process.cache.volume_1.read.failure,string"`
	Proxy_process_cache_volume_1_write_active                          float64 `json:"proxy.process.cache.volume_1.write.active,string"`
	Proxy_process_cache_volume_1_write_success                         float64 `json:"proxy.process.cache.volume_1.write.success,string"`
	Proxy_process_cache_volume_1_write_failure                         float64 `json:"proxy.process.cache.volume_1.write.failure,string"`
	Proxy_process_cache_volume_1_write_backlog_failure                 float64 `json:"proxy.process.cache.volume_1.write.backlog.failure,string"`
	Proxy_process_cache_volume_1_update_active                         float64 `json:"proxy.process.cache.volume_1.update.active,string"`
	Proxy_process_cache_volume_1_update_success                        float64 `json:"proxy.process.cache.volume_1.update.success,string"`
	Proxy_process_cache_volume_1_update_failure                        float64 `json:"proxy.process.cache.volume_1.update.failure,string"`
	Proxy_process_cache_volume_1_remove_active                         float64 `json:"proxy.process.cache.volume_1.remove.active,string"`
	Proxy_process_cache_volume_1_remove_success                        float64 `json:"proxy.process.cache.volume_1.remove.success,string"`
	Proxy_process_cache_volume_1_remove_failure                        float64 `json:"proxy.process.cache.volume_1.remove.failure,string"`
	Proxy_process_cache_volume_1_evacuate_active                       float64 `json:"proxy.process.cache.volume_1.evacuate.active,string"`
	Proxy_process_cache_volume_1_evacuate_success                      float64 `json:"proxy.process.cache.volume_1.evacuate.success,string"`
	Proxy_process_cache_volume_1_evacuate_failure                      float64 `json:"proxy.process.cache.volume_1.evacuate.failure,string"`
	Proxy_process_cache_volume_1_scan_active                           float64 `json:"proxy.process.cache.volume_1.scan.active,string"`
	Proxy_process_cache_volume_1_scan_success                          float64 `json:"proxy.process.cache.volume_1.scan.success,string"`
	Proxy_process_cache_volume_1_scan_failure                          float64 `json:"proxy.process.cache.volume_1.scan.failure,string"`
	Proxy_process_cache_volume_1_direntries_total                      float64 `json:"proxy.process.cache.volume_1.direntries.total,string"`
	Proxy_process_cache_volume_1_direntries_used                       float64 `json:"proxy.process.cache.volume_1.direntries.used,string"`
	Proxy_process_cache_volume_1_directory_collision                   float64 `json:"proxy.process.cache.volume_1.directory_collision,string"`
	Proxy_process_cache_volume_1_frags_per_doc_1                       float64 `json:"proxy.process.cache.volume_1.frags_per_doc.1,string"`
	Proxy_process_cache_volume_1_frags_per_doc_2                       float64 `json:"proxy.process.cache.volume_1.frags_per_doc.2,string"`
	Proxy_process_cache_volume_1_read_busy_success                     float64 `json:"proxy.process.cache.volume_1.read_busy.success,string"`
	Proxy_process_cache_volume_1_read_busy_failure                     float64 `json:"proxy.process.cache.volume_1.read_busy.failure,string"`
	Proxy_process_cache_volume_1_write_bytes_stat                      float64 `json:"proxy.process.cache.volume_1.write_bytes_stat,string"`
	Proxy_process_cache_volume_1_vector_marshals                       float64 `json:"proxy.process.cache.volume_1.vector_marshals,string"`
	Proxy_process_cache_volume_1_hdr_marshals                          float64 `json:"proxy.process.cache.volume_1.hdr_marshals,string"`
	Proxy_process_cache_volume_1_hdr_marshal_bytes                     float64 `json:"proxy.process.cache.volume_1.hdr_marshal_bytes,string"`
	Proxy_process_cache_volume_1_gc_bytes_evacuated                    float64 `json:"proxy.process.cache.volume_1.gc_bytes_evacuated,string"`
	Proxy_process_cache_volume_1_gc_frags_evacuated                    float64 `json:"proxy.process.cache.volume_1.gc_frags_evacuated,string"`
	Proxy_process_cache_volume_1_wrap_count                            float64 `json:"proxy.process.cache.volume_1.wrap_count,string"`
	Proxy_process_cache_volume_1_sync_count                            float64 `json:"proxy.process.cache.volume_1.sync.count,string"`
	Proxy_process_cache_volume_1_sync_bytes                            float64 `json:"proxy.process.cache.volume_1.sync.bytes,string"`
	Proxy_process_cache_volume_1_sync_time                             float64 `json:"proxy.process.cache.volume_1.sync.time,string"`
	Proxy_process_cache_volume_1_span_errors_read                      float64 `json:"proxy.process.cache.volume_1.span.errors.read,string"`
	Proxy_process_cache_volume_1_span_errors_write                     float64 `json:"proxy.process.cache.volume_1.span.errors.write,string"`
	Proxy_process_cache_volume_1_span_failing                          float64 `json:"proxy.process.cache.volume_1.span.failing,string"`
	Proxy_process_cache_volume_1_span_offline                          float64 `json:"proxy.process.cache.volume_1.span.offline,string"`
	Proxy_process_cache_volume_1_span_online                           float64 `json:"proxy.process.cache.volume_1.span.online,string"`
}

func (c TrafficServerCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- up
}

func (c TrafficServerCollector) Collect(ch chan<- prometheus.Metric) {

	// FIXME: need to make these arguments
	var uri string = c.trafficServerScrapeUri
	var sslVerify bool = c.trafficServerSslVerify
	var timeout = time.Duration(c.trafficServerScrapeTimeout) * time.Second
	_, err := url.Parse(uri)

	if err != nil {
		ch <- prometheus.MustNewConstMetric(up, prometheus.GaugeValue, 0)
		return
	}

	body, err := fetchHTTP(uri, sslVerify, timeout)

	// print out the body for debug purposes
	//buf, _ := ioutil.ReadAll(body)
	//rdr1 := ioutil.NopCloser(bytes.NewBuffer(buf))
	//log.Info(rdr1)

	if err != nil {
		ch <- prometheus.MustNewConstMetric(up, prometheus.GaugeValue, 0)
		return
	}

	decoder := json.NewDecoder(body)
	var metrics Metrics
	decode_err := decoder.Decode(&metrics)
	if decode_err != nil {
		ch <- prometheus.MustNewConstMetric(up, prometheus.GaugeValue, 0)
		return
	}

	// This means things are healthy, so we can return an up
	ch <- prometheus.MustNewConstMetric(up, prometheus.GaugeValue, 1)

	// deal with all of our counters
	fields := reflect.TypeOf(metrics.Counters)
	values := reflect.ValueOf(metrics.Counters)
	num := fields.NumField()

	for i := 0; i < num; i++ {
		field := fields.Field(i)
		name := strings.ToLower("trafficserver_" + invalidChars.ReplaceAllLiteralString(field.Name, "_"))
		desc := prometheus.NewDesc(name, "Trafficserver metric "+field.Name, nil, nil)
		value := values.Field(i)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.CounterValue, float64(value.Float()))
	}

	// do the same with the gauges, histograms, and summarys
	// TODO - figure out what metrics are gauges and which are counters
}

func fetchHTTP(uri string, sslVerify bool, timeout time.Duration) (io.Reader, error) {
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: !sslVerify}}
	client := http.Client{
		Timeout:   timeout,
		Transport: tr,
	}

	resp, err := client.Get(uri)

	if err != nil {
		return nil, err
	}
	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP status %d", resp.StatusCode)
	}
	return resp.Body, nil
}

func main() {
	var (
		listenAddress          = kingpin.Flag("web.listen-address", "Address to listen on for web interface and telemetry.").Default(":9548").String()
		metricsPath            = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
		trafficServerScrapeURI = kingpin.Flag("trafficserver.scrape-uri", "URI on which to scrape TrafficServer.").Default("http://localhost/_stats").String()
		trafficServerSSLVerify = kingpin.Flag("trafficserver.ssl-verify", "Flag that enables SSL certificate verification for the scrape URI").Default("true").Bool()
		trafficServerTimeout   = kingpin.Flag("trafficserver.timeout", "Timeout for trying to get stats from TrafficServer.").Default("5").Int()
	)

	log.AddFlags(kingpin.CommandLine)
	kingpin.Version(version.Print("trafficserver_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	log.Infoln("Starting trafficserver_exporter", version.Info())
	log.Infoln("Build context", version.BuildContext())

	log.Infoln("Listening on", *listenAddress)
	log.Infoln("Scraping from", *trafficServerScrapeURI)

	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>Trafficserver Exporter</title></head>
             <body>
             <h1>Trafficserver Exporter</h1>
             <p><a href='` + *metricsPath + `'>Metrics</a></p>
             </body>
             </html>`))
	})

	c := TrafficServerCollector{
		trafficServerScrapeUri:     *trafficServerScrapeURI,
		trafficServerScrapeTimeout: int(*trafficServerTimeout),
		trafficServerSslVerify:     bool(*trafficServerSSLVerify),
	}

	prometheus.MustRegister(c)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
