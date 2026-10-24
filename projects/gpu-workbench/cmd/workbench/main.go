package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"gpu-workbench/internal/workbench"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

func main() {
	if e := run(); e != nil {
		log.Fatal(e)
	}
}
func run() error {
	mode := flag.String("mode", "local-demo", "local-demo, k8s-cpu, k8s-sim, k8s-koord-hami")
	addr := flag.String("listen", "127.0.0.1:8787", "HTTP address")
	data := flag.String("data", ".runtime", "private state directory")
	config := flag.String("kubeconfig", "", "dedicated kubeconfig")
	ns := flag.String("namespace", "gpu-workbench", "dedicated namespace")
	image := flag.String("image", "python:3.12.11-slim-bookworm", "controlled workload image")
	flag.Parse()
	switch *mode {
	case "local-demo", "k8s-cpu", "k8s-sim", "k8s-koord-hami":
	default:
		return fmt.Errorf("unsupported mode")
	}
	if *mode == "k8s-koord-hami" && *image == "python:3.12.11-slim-bookworm" {
		return fmt.Errorf("real GPU mode requires the CUDA workload image; see deploy/real-gpu.md")
	}
	if e := os.MkdirAll(*data, 0700); e != nil {
		return e
	}
	lock, e := os.OpenFile(filepath.Join(*data, *mode+".lock"), os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return e
	}
	defer lock.Close()
	if e = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); e != nil {
		return fmt.Errorf("another process owns this mode: %w", e)
	}
	s, e := workbench.Open(filepath.Join(*data, *mode+".db"))
	if e != nil {
		return e
	}
	defer s.Close()
	credentialPath := filepath.Join(*data, "credentials.json")
	tokens := map[string]string{}
	raw, e := os.ReadFile(credentialPath)
	if os.IsNotExist(e) {
		for _, team := range []string{"alpha", "beta"} {
			b := make([]byte, 24)
			if _, e = rand.Read(b); e != nil {
				return e
			}
			tokens[team] = hex.EncodeToString(b)
		}
		raw, e = json.MarshalIndent(tokens, "", "  ")
		if e != nil {
			return e
		}
		if e = os.WriteFile(credentialPath, raw, 0600); e != nil {
			return e
		}
	} else if e != nil {
		return e
	} else if e = json.Unmarshal(raw, &tokens); e != nil {
		return e
	}
	for _, team := range []string{"alpha", "beta"} {
		if e = s.ConfigureTeam(workbench.Team{ID: team, MaxTasks: 8, MaxMemoryMiB: 32768, MaxCores: 400}, tokens[team]); e != nil {
			return e
		}
	}
	var backend workbench.Backend
	identity := *mode
	if *mode == "local-demo" {
		backend = workbench.NewDemo(s)
	} else {
		var cfg *rest.Config
		if *config != "" {
			cfg, e = clientcmd.BuildConfigFromFlags("", *config)
		} else {
			cfg, e = rest.InClusterConfig()
		}
		if e != nil {
			return e
		}
		cfg.Timeout = 10 * time.Second
		client, e := kubernetes.NewForConfig(cfg)
		if e != nil {
			return e
		}
		dc, e := dynamic.NewForConfig(cfg)
		if e != nil {
			return e
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		namespace, e := client.CoreV1().Namespaces().Get(ctx, *ns, meta.GetOptions{})
		cancel()
		if e != nil {
			return fmt.Errorf("namespace preflight: %w", e)
		}
		identity += "|" + *ns + "|" + string(namespace.UID)
		backend = &workbench.Kube{Client: client, Dynamic: dc, Namespace: *ns, Mode: *mode, Image: *image}
	}
	if e = s.BindBackend(identity); e != nil {
		return e
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	controller := &workbench.Controller{Store: s, Backend: backend}
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				stepCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
				e := controller.Step(stepCtx)
				cancel()
				if e != nil && ctx.Err() == nil {
					log.Printf("controller: %v", e)
				}
			}
		}
	}()
	server := &http.Server{Addr: *addr, Handler: workbench.NewHandler(s, backend, *mode), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second}
	errs := make(chan error, 1)
	go func() { errs <- server.ListenAndServe() }()
	log.Printf("GPU Workbench mode=%s address=http://%s credentials=%s", *mode, *addr, credentialPath)
	select {
	case e = <-errs:
		stop()
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	server.Shutdown(shutdown)
	stop()
	<-done
	if e == http.ErrServerClosed {
		return nil
	}
	return e
}
