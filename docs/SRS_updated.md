# sentinel-agent
# Software Requirements Specification

Version: 1.0
Status: Active
Repository: github.com/Sentinel-Security-Management/sentinel-agent
References: PRD.md (goals and requirements), ARD.md (architecture decisions)

---

## 1. System overview

sentinel-agent is a single long-lived process built as a pre-configured OTel Collector distribution. It reads from the local host environment and writes exclusively to one destination: the OTLP/gRPC endpoint of sentinel-processor, running inside the customer's VPC.

```
host environment                         customer VPC
....................                      ............

hostmetricsreceiver   --+
kubeletstatsreceiver  --+
k8sclusterreceiver    --+
dockerstatsreceiver   --+                sentinel-processor
filelogreceiver       --+--> pipeline -->   OTel Collector
otlpreceiver (OBI)    --+    (process,      (ClickHouse
otlpreceiver (SDK)    --+     stamp,         writer,
                              scrub,         entity
                              batch)         service,
                                             NATS)

no other outbound connections permitted
```

The pipeline inside sentinel-agent runs every signal through five processors in order before export. All subsystems share a single OTel Resource built at startup by the cloud identity resolver. All subsystems export through a single mTLS-authenticated OTLP exporter.

---

## 2. Configuration

All configuration is via environment variables. No configuration file is required to run. sentinelhq provision generates a ConfigMap or environment file with the correct values for the detected environment.

### 2.1 Environment variables

```
SENTINEL_OTLP_ENDPOINT
  type: string
  default: localhost:4317
  description: OTLP/gRPC endpoint of sentinel-processor

SENTINEL_OTLP_TLS
  type: bool
  default: false
  description: enable mTLS on the OTLP connection

SENTINEL_OTLP_CERT_FILE
  type: string
  default: (empty)
  description: path to client certificate for mTLS

SENTINEL_OTLP_KEY_FILE
  type: string
  default: (empty)
  description: path to client private key for mTLS

SENTINEL_OTLP_CA_FILE
  type: string
  default: (empty)
  description: path to CA certificate to verify sentinel-processor

SENTINEL_LOG_PATHS
  type: string (comma-separated glob patterns)
  default: /var/log/*.log
  description: log file paths to tail

SENTINEL_K8S_ENABLED
  type: string (auto | true | false)
  default: auto
  description: enable Kubernetes mode

SENTINEL_DOCKER_ENABLED
  type: string (auto | true | false)
  default: auto
  description: enable Docker mode

SENTINEL_SCRUB_POLICY
  type: string (file path)
  default: /etc/sentinel/sentinel-policy.yaml
  description: path to the PII policy file

SENTINEL_MEMORY_LIMIT_MIB
  type: int
  default: 400
  description: memory limit for the memorylimiterprocessor in MiB

SENTINEL_QUEUE_MAX_SIZE
  type: int
  default: 5000
  description: maximum number of batches in the persistent export queue

SENTINEL_LOG_LEVEL
  type: string (debug | info | warn | error)
  default: info
  description: log level for sentinel-agent's own logs

NODE_NAME
  type: string
  injected by: Kubernetes downward API
  description: k8s.node.name attribute value

POD_NAME
  type: string
  injected by: Kubernetes downward API
  description: k8s.pod.name attribute value

POD_UID
  type: string
  injected by: Kubernetes downward API
  description: k8s.pod.uid attribute value

POD_NAMESPACE
  type: string
  injected by: Kubernetes downward API
  description: k8s.namespace.name attribute value

CONTAINER_NAME
  type: string
  injected by: Kubernetes downward API
  description: k8s.container.name attribute value
```

### 2.2 Validation rules

SENTINEL_OTLP_ENDPOINT must be a valid host:port string.
If SENTINEL_OTLP_TLS is true, SENTINEL_OTLP_CERT_FILE and SENTINEL_OTLP_KEY_FILE must be non-empty and the files must exist.
SENTINEL_LOG_LEVEL must be one of: debug, info, warn, error.
SENTINEL_K8S_ENABLED and SENTINEL_DOCKER_ENABLED must be one of: auto, true, false.
SENTINEL_MEMORY_LIMIT_MIB must be a positive integer greater than 100.
SENTINEL_QUEUE_MAX_SIZE must be a positive integer.

Validation failures must produce a human-readable error message and exit with code 1 before any receivers are started.

---

## 3. Package specifications

### 3.1 Package: config

Purpose: single source of truth for all configuration. Reads from environment variables. Validates at startup. Returns a typed Config struct.

```go
type Config struct {
    OTLPEndpoint      string
    OTLPTLSEnabled    bool
    OTLPCertFile      string
    OTLPKeyFile       string
    OTLPCAFile        string
    LogPaths          []string
    K8sEnabled        string
    DockerEnabled     string
    ScrubPolicyPath   string
    MemoryLimitMiB    int
    QueueMaxSize      int
    LogLevel          string
    K8s               K8sMetadata
}

type K8sMetadata struct {
    NodeName      string
    PodName       string
    PodUID        string
    NamespaceName string
    ContainerName string
}

func Load() (*Config, error)
```

Load reads all environment variables, applies defaults, validates, and returns. Returns a descriptive error on any validation failure. Never panics.

### 3.2 Package: identity

Purpose: resolves cloud and host identity at startup. Returns a populated OTel Resource. Runs once. Result is shared across all subsystems.

```go
type CloudIdentity struct {
    Provider    string  // aws | gcp | azure | bare_metal
    Region      string
    AccountID   string  // AWS only
    HostID      string
    HostName    string
    HostType    string  // instance type, AWS only
}

func Detect(ctx context.Context) *CloudIdentity
func (c *CloudIdentity) ToOTelResource(k8s *config.K8sMetadata) *sdkresource.Resource
```

Detect runs probes sequentially. Each probe has a context with an 80ms deadline derived from the parent context. The parent context deadline is 500ms from call time. First successful probe returns immediately. All probes timing out returns a bare_metal identity. Detect never returns nil. Detect never returns an error.

#### AWS probe

```
PUT http://169.254.169.254/latest/api/token
  Header: X-aws-ec2-metadata-token-ttl-seconds: 60
  Timeout: 80ms

On 200:
  token = response body
  GET http://169.254.169.254/latest/meta-data/placement/region
    Header: X-aws-ec2-metadata-token: {token}
  GET http://169.254.169.254/latest/meta-data/instance-id
    Header: X-aws-ec2-metadata-token: {token}
  GET http://169.254.169.254/latest/meta-data/instance-type
    Header: X-aws-ec2-metadata-token: {token}
  GET http://169.254.169.254/latest/meta-data/iam/info
    Header: X-aws-ec2-metadata-token: {token}
    parse AccountId from JSON response

Sets: Provider=aws, Region, HostID, HostType, AccountID
```

#### GCP probe

```
GET http://metadata.google.internal/computeMetadata/v1/instance/id
  Header: Metadata-Flavor: Google
  Timeout: 80ms

On 200:
  GET http://metadata.google.internal/computeMetadata/v1/instance/zone
    Header: Metadata-Flavor: Google
  zone format: projects/{project}/zones/{region}-{zone-letter}
  extract region by stripping trailing dash and letter

Sets: Provider=gcp, HostID, Region
```

#### Azure probe

```
GET http://169.254.169.254/metadata/instance?api-version=2021-02-01
  Header: Metadata: true
  Timeout: 80ms

On 200:
  parse JSON response body
  extract compute.location as Region
  extract compute.vmId as HostID
  extract compute.name as HostName
  extract compute.vmSize as HostType

Sets: Provider=azure, Region, HostID, HostName, HostType
```

#### Bare metal fallback

```
Sets: Provider=bare_metal
      HostName = os.Hostname() result
      Region = ""
      HostID = ""
```

#### OTel Resource construction

ToOTelResource constructs an OTel SDK Resource with all non-empty attributes. It always includes cloud.provider and host.name. All other attributes are included only when non-empty.

```
Resource attributes:
  cloud.provider        always present
  host.name             always present
  cloud.region          when non-empty
  host.id               when non-empty
  cloud.account.id      when non-empty (AWS)
  host.type             when non-empty (AWS, Azure)
  k8s.node.name         when K8sMetadata.NodeName non-empty
  k8s.pod.name          when K8sMetadata.PodName non-empty
  k8s.pod.uid           when K8sMetadata.PodUID non-empty
  k8s.namespace.name    when K8sMetadata.NamespaceName non-empty
  k8s.container.name    when K8sMetadata.ContainerName non-empty
```

#### Test acceptance criteria

TestDetect_AWSMock: mock IMDSv2 server returns valid token and metadata. Detect returns Provider=aws with correct region and instance-id. Completes in under 200ms.

TestDetect_GCPMock: mock GCP metadata server returns valid instance ID and zone. Detect returns Provider=gcp with correct host ID and region parsed from zone string.

TestDetect_AzureMock: mock Azure IMDS returns valid JSON. Detect returns Provider=azure with correct location, vmId.

TestDetect_BareMetalFallback: all probes return connection refused. Detect returns Provider=bare_metal with non-empty HostName. Completes in under 600ms.

TestDetect_Timeout: all mock servers delay 200ms before responding. Detect returns Provider=bare_metal in under 600ms total.

TestDetect_PartialFailure: AWS probe returns 404, GCP probe returns valid response. Detect returns Provider=gcp.

TestToOTelResource_AllAttributes: CloudIdentity with all fields populated plus K8sMetadata with all fields populated. Resource contains all eleven attributes.

TestToOTelResource_BareMetalNoK8s: bare_metal identity with empty K8sMetadata. Resource contains only cloud.provider and host.name.

### 3.3 Package: pipeline

Purpose: constructs and runs the OTel Collector pipeline. Wires receivers, processors, and exporters according to the detected environment. This is configuration code, not business logic.

```go
type Pipeline struct {
    cfg      *config.Config
    resource *sdkresource.Resource
}

func New(cfg *config.Config, resource *sdkresource.Resource) *Pipeline

// Build constructs the OTel Collector service graph.
// Returns an error if any component fails to initialise.
func (p *Pipeline) Build(ctx context.Context) (*otelcol.Collector, error)

// Run starts the collector. Blocks until ctx is cancelled.
func (p *Pipeline) Run(ctx context.Context) error
```

Build activates receivers based on environment detection:

```
if K8sEnabled == true OR (K8sEnabled == auto AND /var/run/secrets/kubernetes.io/serviceaccount exists):
  activate kubeletstatsreceiver
  activate k8sclusterreceiver
  set filelogreceiver include = ["/var/log/pods/*/*/*.log"]
  set filelogreceiver operators to parse Kubernetes log format

else if DockerEnabled == true OR (DockerEnabled == auto AND /var/run/docker.sock exists):
  activate dockerstatsreceiver
  activate hostmetricsreceiver
  set filelogreceiver include = ["/var/lib/docker/containers/*/*-json.log"]
  set filelogreceiver operators to parse Docker JSON log format

else:
  activate hostmetricsreceiver
  set filelogreceiver include = SENTINEL_LOG_PATHS

always activate:
  otlpreceiver on localhost:4317
```

Processor order is always:
1. memorylimiterprocessor (limit from SENTINEL_MEMORY_LIMIT_MIB)
2. resourcedetectionprocessor (using identity.CloudIdentity)
3. k8sattributesprocessor (when in Kubernetes mode only)
4. sentinel-scrub processor (using SENTINEL_SCRUB_POLICY)
5. batchprocessor (send_batch_size=8192, timeout=5s)

Exporter is always:
otlpexporter with endpoint from SENTINEL_OTLP_ENDPOINT, mTLS from cert files when SENTINEL_OTLP_TLS=true, retry and persistent queue configuration.

### 3.4 Package: scrub

Purpose: the sentinel-scrub OTel Collector processor. Reads sentinel-policy.yaml and enforces the PII policy on every signal passing through the pipeline.

```go
type Policy struct {
    Fields           []FieldRule           `yaml:"fields"`
    RuntimeDetection RuntimeDetectionRule  `yaml:"runtime_detection"`
}

type FieldRule struct {
    Path       string `yaml:"path"`
    Action     string `yaml:"action"`     // hash | drop | keep
    Confidence string `yaml:"confidence"` // high | medium | low | review_required
}

type RuntimeDetectionRule struct {
    Enabled        bool     `yaml:"enabled"`
    Patterns       []string `yaml:"patterns"` // email | phone | credit_card | jwt | ssn | ip_address
    FallbackAction string   `yaml:"fallback_action"` // hash | drop
}

func LoadPolicy(path string) (*Policy, error)

type Processor struct {
    policy *Policy
}

func NewProcessor(policy *Policy) *Processor

// Process enforces the policy on a pdata.Traces, pdata.Metrics, or pdata.Logs.
// Returns the modified signal. Never returns an error.
// Errors in individual attribute processing are logged at WARN and skipped.
func (p *Processor) ProcessTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error)
func (p *Processor) ProcessMetrics(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error)
func (p *Processor) ProcessLogs(ctx context.Context, ld plog.Logs) (plog.Logs, error)
```

Hash function: SHA-256 of the UTF-8 encoded string value, encoded as lowercase hex. The hash is deterministic. The same input always produces the same output, preserving correlation across signals without exposing the value.

Runtime detection patterns:

```
email:       RFC 5321 local-part @ domain pattern
             [a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}

credit_card: 13-19 digit sequences passing Luhn algorithm check
             preceded and followed by non-digit characters

jwt:         three base64url segments separated by dots
             eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+

phone:       E.164 international format and common regional formats
             (\+?[1-9]\d{6,14})

ssn:         US Social Security Number format
             \b\d{3}-\d{2}-\d{4}\b

ip_address:  IPv4 addresses in attribute string values
             \b(?:\d{1,3}\.){3}\d{1,3}\b
```

Runtime detection is applied to string attribute values regardless of attribute name. When a pattern matches and fallback_action is hash, the matched substring is replaced with its SHA-256 hash. When fallback_action is drop, the entire attribute is removed.

Policy file hot reload: when sentinel-agent receives SIGHUP, Processor reloads sentinel-policy.yaml without stopping the pipeline. Signals in flight at the moment of reload complete with the previous policy. Signals entering the processor after the reload complete with the new policy. If the new policy fails to parse, the previous policy remains active and a WARN log is emitted.

#### Test acceptance criteria

TestLoadPolicy_Valid: valid sentinel-policy.yaml parses correctly into Policy struct.
TestLoadPolicy_Missing: returns error when file does not exist.
TestLoadPolicy_InvalidYAML: returns error with line number context.

TestProcessTraces_HashAction: trace span with user.id=usr_8823, policy hashes user.id. Output span has user.id=SHA256("usr_8823").
TestProcessTraces_DropAction: trace span with http.request.body=..., policy drops it. Output span has no http.request.body attribute.
TestProcessTraces_KeepAction: trace span with service.name=payment, policy keeps it. Output span has service.name=payment unchanged.
TestProcessTraces_RuntimeEmail: trace span with custom_field="contact john@example.com for details", runtime detection enabled for email. Output has custom_field with john@example.com replaced by its hash.
TestProcessTraces_RuntimeCreditCard: span attribute value containing a valid Luhn-passing number. Output has number replaced by hash.
TestProcessTraces_RuntimeJWT: span attribute value containing a valid JWT pattern. Output has JWT replaced or attribute dropped per fallback_action.
TestProcessLogs_PII: log body containing email address. Output log body has email replaced by hash.
TestProcessor_HotReload: processor running, send SIGHUP, verify new policy takes effect within one scrape interval without restart.
TestProcessor_HotReloadInvalidPolicy: processor running with valid policy, reload with invalid YAML. Verify original policy still active and WARN was logged.

### 3.5 Package: cmd/agent

Purpose: entry point. Wires all packages. Manages process lifecycle. Handles OS signals.

```go
func main()
```

Startup sequence:

```
1. config.Load()
     on error: log error, os.Exit(1)

2. identity.Detect(ctx)
     timeout: 500ms
     never fails, always returns a result

3. resource = identity.ToOTelResource(cfg.K8s)

4. policy = scrub.LoadPolicy(cfg.ScrubPolicyPath)
     on error: log error, os.Exit(1)
     policy file must exist at startup

5. pipeline.New(cfg, resource).Build(ctx)
     on error: log error, os.Exit(1)

6. signal.NotifyContext(SIGTERM, SIGINT)
     cancels ctx on signal

7. pipeline.Run(ctx)
     blocks until ctx cancelled

8. exporter.Shutdown(ctx with 30s deadline)
     drains in-flight signals

9. os.Exit(0)
```

Error handling: any goroutine returning a non-recoverable error cancels the root context via errgroup. This triggers shutdown of all other goroutines. Panics are caught at the top level, logged, and result in os.Exit(1).

SIGTERM handling: cancel the root context. This stops all receivers from accepting new signals, drains the processor pipeline, flushes the exporter queue. Force exit after 30 seconds if drain is not complete.

SIGHUP handling: reload sentinel-policy.yaml in the scrub processor without restarting.

#### Test acceptance criteria

TestMain_StartsAndExports: integration test. Agent starts with mock OTLP server as sentinel-processor. Within 20 seconds, at least one metric record arrives at the mock server with cloud.provider and host.name resource attributes present.
TestMain_GracefulShutdown: agent starts, SIGTERM sent, process exits 0 within 35 seconds, all pre-shutdown signals received by mock server.
TestMain_SurvivesOutage: agent starts, mock server stops for 5 minutes, mock server restarts, all queued signals delivered without agent restart.
TestMain_ConfigValidationFails: SENTINEL_OTLP_TLS=true with missing cert file. Process exits 1 with a clear error message before starting any receivers.
TestMain_PolicyMissing: SENTINEL_SCRUB_POLICY points to non-existent file. Process exits 1 with a clear error message.

---

## 4. Receiver specifications

### 4.1 hostmetricsreceiver

Used in: Docker mode, VM mode, bare metal mode.
Collection interval: 15 seconds (configurable via OTel Collector config, not a separate env var).

Required scrapers and output metrics:

```
cpu scraper:
  system.cpu.utilization
    type: gauge
    unit: 1 (ratio 0.0 to 1.0)
    attributes: cpu (cpu0, cpu1, ..., total), state (user, system, idle, iowait)

memory scraper:
  system.memory.usage
    type: gauge
    unit: By
    attributes: state (used, free, cached, available, slab_reclaimable)
  system.memory.utilization
    type: gauge
    unit: 1 (ratio 0.0 to 1.0)
    attributes: state (used, free, cached, available)

disk scraper:
  system.disk.io
    type: sum (monotonic)
    unit: By
    attributes: device (sda, nvme0n1, ...), direction (read, write)
  system.disk.operations
    type: sum (monotonic)
    unit: {operations}
    attributes: device, direction (read, write)

filesystem scraper:
  system.filesystem.usage
    type: gauge
    unit: By
    attributes: device, mountpoint, type, state (used, free, reserved)
  system.filesystem.utilization
    type: gauge
    unit: 1
    attributes: device, mountpoint, type

network scraper:
  system.network.io
    type: sum (monotonic)
    unit: By
    attributes: device (eth0, lo, ...), direction (transmit, receive)
  system.network.errors
    type: sum (monotonic)
    unit: {errors}
    attributes: device, direction

load scraper:
  system.cpu.load_average.1m
  system.cpu.load_average.5m
  system.cpu.load_average.15m
    type: gauge
    unit: {processes}
```

### 4.2 kubeletstatsreceiver

Used in: Kubernetes mode.
Collection interval: 15 seconds.
Auth: ServiceAccount token from /var/run/secrets/kubernetes.io/serviceaccount.

Key metrics emitted:

```
k8s.node.cpu.utilization
k8s.node.memory.usage
k8s.node.filesystem.usage
k8s.node.network.io
k8s.pod.cpu.utilization
k8s.pod.memory.usage
k8s.pod.network.io
k8s.container.cpu.utilization
k8s.container.memory.usage
```

### 4.3 k8sclusterreceiver

Used in: Kubernetes mode. Runs only on one agent in the cluster (leader election via ConfigMap lock). All others are passive.
Collection interval: 30 seconds.

Key metrics and events emitted:

```
k8s.deployment.available
k8s.deployment.desired
k8s.pod.phase (as metric with phase attribute: running, pending, failed, succeeded)
k8s.replicaset.available
k8s.replicaset.desired
k8s.namespace.phase
```

K8s events emitted as OTel log records with k8s.event.reason, k8s.event.message, k8s.event.type (Normal, Warning) attributes.

### 4.4 dockerstatsreceiver

Used in: Docker mode.
Collection interval: 15 seconds.
Requires: docker.sock mounted as a volume.

Key metrics emitted:

```
container.cpu.utilization
container.memory.usage.total
container.memory.percent
container.network.io.usage.rx_bytes
container.network.io.usage.tx_bytes
```

Additional attributes stamped: container.name, container.id, container.image.name.

### 4.5 filelogreceiver

Used in: all modes with different include paths.

Log record format:

```
Body:               raw log line string, UTF-8, trimmed of trailing newline
Timestamp:          time of line read
ObservedTimestamp:  same as Timestamp
SeverityText:       empty (sentinel-processor parses severity in its pipeline)
SeverityNumber:     UNSPECIFIED
Attributes:
  log.file.path:    absolute path of the source file
  log.file.name:    filename only
Resource:           shared OTel Resource from identity.ToOTelResource
```

Offset tracking: the receiver must persist the last-read byte offset for each file. On restart, reading resumes from the persisted offset. If the state file is missing, reading starts from the end of the file, not the beginning.

Rotation handling: the receiver must detect log rotation via file truncation and inode change. On detection it closes the current file handle and reopens.

### 4.6 otlpreceiver

Used in: all modes.
Listens on: localhost:4317 (gRPC), localhost:55681 (HTTP, optional).
Accepts: metrics, logs, and traces from OBI and sentinel-sdk.
No authentication required on this receiver. It listens only on localhost, not on any external interface.

---

## 5. Data format reference

### 5.1 Resource attributes

Every signal exported by sentinel-agent carries these resource attributes:

```
cloud.provider       string  always present  aws | gcp | azure | bare_metal
host.name            string  always present  OS hostname
cloud.region         string  cloud only      us-east-1 | europe-west1 | etc.
host.id              string  cloud only      EC2 instance ID, GCP instance ID, Azure VM ID
cloud.account.id     string  AWS only        AWS account ID
host.type            string  AWS/Azure only  instance type
k8s.node.name        string  K8s only        from NODE_NAME downward API
k8s.pod.name         string  K8s only        from POD_NAME downward API
k8s.pod.uid          string  K8s only        from POD_UID downward API
k8s.namespace.name   string  K8s only        from POD_NAMESPACE downward API
k8s.container.name   string  K8s only        from CONTAINER_NAME downward API
```

### 5.2 Export schema

sentinel-agent exports three OTLP signal types to the same gRPC endpoint:

```
MetricsService/Export  -> all metric records
LogsService/Export     -> all log records
TraceService/Export    -> all trace spans

All signals include ResourceMetrics/ResourceLogs/ResourceSpans
with the full resource attribute set from section 5.1
```

### 5.3 sentinel-policy.yaml schema

```yaml
version: 1
generated_by: sentinel-scrub-scanner | manual

fields:
  - path: "attribute.path"       # dot-separated OTel attribute path
    action: hash | drop | keep
    confidence: high | medium | low | review_required
    detected_in: ["file:line"]   # scanner-generated only

runtime_detection:
  enabled: true
  patterns:
    - email
    - phone
    - credit_card
    - jwt
    - ssn
    - ip_address
  fallback_action: hash | drop
```

---

## 6. RBAC requirements

For Kubernetes deployments, sentinel-agent requires the following ClusterRole:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: sentinel-agent
rules:
  - apiGroups: [""]
    resources: ["nodes", "nodes/stats", "nodes/metrics", "pods"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["apps"]
    resources: ["replicasets", "deployments", "daemonsets", "statefulsets"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["events"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["configmaps"]
    verbs: ["get", "create", "update"]
    resourceNames: ["sentinel-agent-leader"]
```

The configmaps permission on sentinel-agent-leader is required for k8sclusterreceiver leader election.

---

## 7. Build requirements

### 7.1 Binary

The sentinel-agent binary must compile with CGO_ENABLED=0 for all code. The binary must be statically linked. It must run on Linux amd64 and arm64. It must not depend on any shared libraries at runtime.

### 7.2 Docker image

Base image: gcr.io/distroless/static for production builds. scratch is acceptable but distroless provides the /etc/ssl/certs CA bundle for HTTPS connections.

Non-root user: UID 65534 (nobody). The DaemonSet security context requests the required capabilities explicitly rather than running as root.

Required Linux capabilities for sentinel-agent:

```
CAP_DAC_READ_SEARCH   for reading log files not owned by the agent user
CAP_SYS_PTRACE        required by hostmetricsreceiver for process metrics
```

Image size must be under 40MB.

### 7.3 CI checks

Every PR must pass all of the following before merge:

```
go build ./...                    linux/amd64 and linux/arm64
go vet ./...                      zero warnings
golangci-lint run                 zero errors with project .golangci.yaml
go test -race ./...               zero failures
go test -coverprofile=coverage.out ./...   coverage >= 80% per package
```

### 7.4 Release artifacts

Each tagged release produces:

```
sentinel-agent-linux-amd64        statically compiled binary
sentinel-agent-linux-arm64        statically compiled binary
sentinel-agent-linux-amd64.sha256 checksum
sentinel-agent-linux-arm64.sha256 checksum
ghcr.io/sentinel-security-management/sentinel-agent:{version}   multi-arch image
```

---

## 8. File layout

```
sentinel-agent/
  cmd/
    agent/
      main.go            entry point, signal handling, lifecycle
  config/
    config.go            Config struct, Load(), validation
    config_test.go
  identity/
    detector.go          CloudIdentity, Detect(), ToOTelResource()
    detector_test.go
  pipeline/
    pipeline.go          Pipeline, Build(), Run()
    pipeline_test.go
    receivers.go         receiver configuration per environment
    processors.go        processor configuration
    exporters.go         exporter configuration with mTLS
  scrub/
    processor.go         Processor, ProcessTraces/Metrics/Logs
    policy.go            Policy, FieldRule, LoadPolicy()
    patterns.go          runtime detection regex patterns
    processor_test.go
    policy_test.go
  go.mod
  go.sum
  Dockerfile
  deploy/
    daemonset.yaml
    rbac.yaml
    configmap-example.yaml
  LICENSE
  README.md
  CONTRIBUTING.md
  PRD.md
  SRS.md
  ARD.md
```
