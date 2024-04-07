# HDNS (DNS to DOT Proxy)
HDNS is a DNS to DNS over TLS (DoT) proxy that can handle both TCP and UDP DNS queries. It can be implemented either as a standalone service or within a Kubernetes infrastructure using the sidecar approach. Upstream DoT servers and their SPKI can be configured using the HDNS config file (stored as a ConfigMap in Kubernetes) . Additionally, HDNS can block domains specified in the config file or through Custom Resource Definitions (CRDs) in a Kubernetes environment.

## INSTALLATION

### 1 ) generate SPKI
#### get the latest SPKI pin with this command(replace 8.8.4.4 if needed)
#### The SPKI is checked for every upstream connection
```
echo | openssl s_client -connect 8.8.4.4:853 2>/dev/null | openssl x509 -pubkey -noout | openssl pkey -pubin -outform der | openssl dgst -sha256 -binary | openssl enc -base64
```
### 2A) non kubenetes environment
#### Generate an hdns.cfg file based on the provided template and place it in the directory /etc/hdns/hdns.cfg.
```
{
    "service_host" : "127.0.0.1",
    "service_port" : "53",
    "service_proto": "all",// can be tcp, udp or all
    "sidecar_injection": "true",
    "dns_host" : "9.9.9.9",
    "dns_port" : "853",
    "dns_spki" : "/SlsviBkb05Y/8XiKF9+CZsgCtrqPQk5bh47o0R3/Cg=",
    "block_domains" : ["google.com"] // you can block unwanted domains
}
```
Any changes to the hdns configuration file (e.g., DNS SPKI) are detected at runtime.

#### Compile the hdns binary or use the pre-built binary located at hdns-service directory
```
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o hdns .
```
#### execute hdns binary
```
./hdns
```

### 2B )  kubenetes environment
#### build the hdns docker image
```
cd hdns-service
docker build . -t hdns:v0.0.1
```
#### build the admission-webhook docker image
```
cd admission-webhook
docker build . -t admission-webhook:v0.0.1  
```
#### install helm chart
```
cd helm-charts
helm package webhook
helm install hdns hdns-webhook-0.0.1.tgz -n hdns  --create-namespace
```
The hdns configuration file is stored in the 'hdns-cm' ConfigMap

#### create a test pod
```
apiVersion: v1
kind: Pod
metadata:
  name: busybox
  namespace: hdns
  labels:
    DNSProxy: "true"
spec:
  containers:
  - name: alpine 
    image: busybox:latest
    command: ["sleep", "3600"] 
  restartPolicy: Never
```

## SCHEMA

### All DNS queries are routed to the sidecar, which then proxies the queries over DNS on TLS (DoT)
The mutating webhook adds an hdns sidecar to each pod containing the label 'DNSProxy'.
```
+===============+     +=================+   +==================+
|    Pod1       |     |      Pod2       |   |      Pod3        |
| Microservice1 |     |  Microservice2  |   |   Microservice3  |
|---------------|     |-----------------|   |------------------|
|               |     |                 |   |                  |  
| hdns sidecar  |     |  hdns sidecar   |   |  hdns sidecar    |           
+===============+     +=================+   +==================+
        ||                     ||                     ||
        ||                     ||                     ||
        ||                     ||                     ||
        ||                     ||                     ||
        ||           +====================+           ||
        ||           |     Pod4           |           ||
        ||===========|  Mutating Webhook  |============
                     | (sidecar injection)|
                     +====================+
```

## hdns CRD

block unwanted domains using CRD

install 
1 - 
```
cd hdns-operator
docker build . -t hdns-operator:v0.0.1  
```
2 - install helm chart
```
cd helm-charts
helm package operator
helm install hdns hdns-operator-0.0.1.tgz -n hdns  --create 
```

```
apiVersion: hdns.io/v1alpha1
kind: DNSBlock
metadata:
  name: test
  namespace: hdns
spec:
  domains:
    - google.com
    - youtube.com
    - facebook.com
```

## Improvements
### 0) implementation of a connection pool for upstream connections (performance)
### 1) rate limit number of requests 
### 2) some firewall for inspecting dns queries
### 3) support for multiple upstreams and loadbalancing between them
### 4) DNS Cache
### 5) support for DNS Over TLSProxy
### 6) Block request type ANY
### 7) Block IPv6 requests
### 8) query monitoring and logging
### 9) forwarding queries for specific domains to some specific upstreams(or based on location)