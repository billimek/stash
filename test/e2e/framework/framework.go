package framework

import (
	"path/filepath"

	"github.com/appscode/go/crypto/rand"
	cs "github.com/appscode/stash/client/clientset/versioned"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ka "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	"kmodules.xyz/client-go/tools/certstore"
)

type Framework struct {
	KubeClient   kubernetes.Interface
	StashClient  cs.Interface
	KAClient     ka.Interface
	namespace    string
	CertStore    *certstore.CertStore
	ClientConfig *rest.Config
	StorageClass string
}

func New(kubeClient kubernetes.Interface, extClient cs.Interface, kaClient ka.Interface, clientConfig *rest.Config, storageClass string) *Framework {
	store, err := certstore.NewCertStore(afero.NewMemMapFs(), filepath.Join("", "pki"))
	Expect(err).NotTo(HaveOccurred())

	err = store.InitCA()
	Expect(err).NotTo(HaveOccurred())

	return &Framework{
		KubeClient:   kubeClient,
		StashClient:  extClient,
		KAClient:     kaClient,
		namespace:    rand.WithUniqSuffix("test-stash"),
		CertStore:    store,
		ClientConfig: clientConfig,
		StorageClass: storageClass,
	}
}

func (f *Framework) Invoke() *Invocation {
	return &Invocation{
		Framework: f,
		app:       rand.WithUniqSuffix("stash-e2e"),
	}
}

func (f *Invocation) AppLabel() string {
	return "app=" + f.app
}

func (f *Invocation) App() string {
	return f.app
}

type Invocation struct {
	*Framework
	app string
}
