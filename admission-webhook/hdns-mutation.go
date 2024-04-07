package main

import (
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"time"

	admissionv1 "k8s.io/api/admission/v1"

	"bytes"

	v1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func handleRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "OK %q", html.EscapeString(r.URL.Path))
}

func handleMutate(w http.ResponseWriter, r *http.Request) {
	admReview, err := admissionReviewFromRequest(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("error getting admission review from request: %v", err)
		return
	}

	admResp, err := admissionResponseFromReview(admReview)

	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	var admissionReviewResponse admissionv1.AdmissionReview
	admissionReviewResponse.Response = admResp
	admissionReviewResponse.SetGroupVersionKind(admReview.GroupVersionKind())
	admissionReviewResponse.Response.UID = admReview.Request.UID

	resp, err := json.Marshal(admissionReviewResponse)
	if err != nil {
		msg := fmt.Errorf("error marshaling response: %v", err)
		log.Println(msg)
		w.WriteHeader(500)
		w.Write([]byte(msg.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	log.Printf("allowing pod as %v", string(resp))
	w.Write(resp)
}

func main() {
	log.Println("starting webhook")

	mux := http.NewServeMux()

	mux.HandleFunc("/", handleRoot)
	mux.HandleFunc("/mutate", handleMutate)

	s := &http.Server{
		Addr:           ":8443",
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	log.Fatal(s.ListenAndServeTLS("/tmp/certs/tls.crt", "/tmp/certs/tls.key"))
}

func admissionReviewFromRequest(r *http.Request) (*admissionv1.AdmissionReview, error) {
	if r.Header.Get("Content-Type") != "application/json" {
		return nil, fmt.Errorf("expected application/json content-type")
	}

	admissionReviewRequest := &admissionv1.AdmissionReview{}

	err := json.NewDecoder(r.Body).Decode(&admissionReviewRequest)
	if err != nil {
		return nil, err
	}
	return admissionReviewRequest, nil
}

func admissionResponseFromReview(admReview *admissionv1.AdmissionReview) (*admissionv1.AdmissionResponse, error) {
	podResource := metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	if admReview.Request.Resource != podResource {
		err := fmt.Errorf("did not receive pod, got %s", admReview.Request.Resource.Resource)
		return nil, err
	}

	admissionResponse := &admissionv1.AdmissionResponse{}

	rawRequest := admReview.Request.Object.Raw
	pod := corev1.Pod{}

	err := json.NewDecoder(bytes.NewReader(rawRequest)).Decode(&pod)
	if err != nil {
		err := fmt.Errorf("error decoding raw pod: %v", err)
		return nil, err
	}

	var patch string
	patchType := v1.PatchTypeJSONPatch

	log.Println("pod has following labels", pod.Labels)
	if _, ok := pod.Labels["DNSProxy"]; ok {
		patch = `[{"op":"add","path":"/spec/containers/1","value":{"image":"hdns:v0.0.1", "imagePullPolicy":"IfNotPresent","name":"hdns-sidecar", "volumeMounts": [{"name": "config-volume", "mountPath": "/etc/hdns"}]}},{"op":"add", "path":"/spec/dnsPolicy", "value": "ClusterFirst"}, {"op":"add", "path":"/spec/dnsConfig", "value":{"nameservers":["127.0.0.1"]}}, {"op":"add", "path":"/spec/volumes/1", "value":{"name":"config-volume", "configMap":{"name":"hdns-cm"}}}]`
	}

	admissionResponse.Allowed = true
	if patch != "" {
		log.Println("patching the pod with:", patch)
		admissionResponse.PatchType = &patchType
		admissionResponse.Patch = []byte(patch)
	}

	return admissionResponse, nil
}
