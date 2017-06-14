package test

import (
	"fmt"

	"github.com/appscode/log"
	rapi "github.com/appscode/restik/api"
	"github.com/appscode/restik/client/clientset"
	"github.com/appscode/restik/pkg/controller"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/apis/apps"
	"k8s.io/kubernetes/pkg/apis/extensions"
)

var namespace string
var podTemplate = &api.PodTemplateSpec{
	ObjectMeta: api.ObjectMeta{
		Name: "nginx",
		Labels: map[string]string{
			"app": "nginx",
		},
	},
	Spec: api.PodSpec{
		Containers: []api.Container{
			{
				Name:  "nginx",
				Image: "nginx",
				VolumeMounts: []api.VolumeMount{
					{
						Name:      "test-volume",
						MountPath: "/source_path",
					},
				},
			},
		},
		Volumes: []api.Volume{
			{
				Name: "test-volume",
				VolumeSource: api.VolumeSource{
					EmptyDir: &api.EmptyDirVolumeSource{},
				},
			},
		},
	},
}

func createTestNamespace(watcher *controller.Controller, name string) error {
	namespace = name
	ns := &api.Namespace{
		ObjectMeta: api.ObjectMeta{
			Name: name,
		},
	}
	_, err := watcher.Clientset.Core().Namespaces().Create(ns)
	return err
}

func deleteTestNamespace(watcher *controller.Controller, name string) {
	if err := watcher.Clientset.Core().Namespaces().Delete(name, &api.DeleteOptions{}); err != nil {
		fmt.Println(err)
	}
}

func createReplicationController(watcher *controller.Controller, name string, backupName string) error {
	kubeClient := watcher.Clientset
	rc := &api.ReplicationController{
		TypeMeta: unversioned.TypeMeta{
			APIVersion: "v1",
			Kind:       "ReplicationController",
		},
		ObjectMeta: api.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				controller.BackupConfig: backupName,
			},
		},
		Spec: api.ReplicationControllerSpec{
			Replicas: 1,
			Template: podTemplate,
		},
	}
	_, err := kubeClient.Core().ReplicationControllers(namespace).Create(rc)
	return err
}

func deleteReplicationController(watcher *controller.Controller, name string) {
	if err := watcher.Clientset.Core().ReplicationControllers(namespace).Delete(name, &api.DeleteOptions{}); err != nil {
		log.Errorln(err)
	}
}

func createSecret(watcher *controller.Controller, name string) error {
	secret := &api.Secret{
		TypeMeta: unversioned.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: api.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"password": []byte("appscode"),
		},
	}
	_, err := watcher.Clientset.Core().Secrets(namespace).Create(secret)
	return err
}

func deleteSecret(watcher *controller.Controller, name string) {
	if err := watcher.Clientset.Core().Secrets(namespace).Delete(name, &api.DeleteOptions{}); err != nil {
		log.Errorln(err)
	}
}

func createRestik(watcher *controller.Controller, backupName string, secretName string) error {
	restik := &rapi.Restik{
		TypeMeta: unversioned.TypeMeta{
			APIVersion: "backup.appscode.com/v1alpha1",
			Kind:       clientset.ResourceKindRestik,
		},
		ObjectMeta: api.ObjectMeta{
			Name:      backupName,
			Namespace: namespace,
		},
		Spec: rapi.RestikSpec{
			Source: rapi.Source{
				Path:       "/source_path",
				VolumeName: "test-volume",
			},
			Schedule: "* * * * * *",
			Destination: rapi.Destination{
				Path:                 "/repo_path",
				RepositorySecretName: secretName,
				Volume: api.Volume{
					Name: "restik-vol",
					VolumeSource: api.VolumeSource{
						EmptyDir: &api.EmptyDirVolumeSource{},
					},
				},
			},
			RetentionPolicy: rapi.RetentionPolicy{
				KeepLastSnapshots: 5,
			},
		},
	}
	_, err := watcher.ExtClientset.Restiks(namespace).Create(restik)
	return err
}

func deleteRestik(watcher *controller.Controller, restikName string) error {
	return watcher.ExtClientset.Restiks(namespace).Delete(restikName, nil)
}

func createReplicaset(watcher *controller.Controller, name string, restikName string) error {
	replicaset := &extensions.ReplicaSet{
		ObjectMeta: api.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				controller.BackupConfig: restikName,
			},
		},
		Spec: extensions.ReplicaSetSpec{
			Replicas: 1,
			Template: *podTemplate,
			Selector: &unversioned.LabelSelector{
				MatchLabels: map[string]string{
					"app": "nginx",
				},
			},
		},
	}
	_, err := watcher.Clientset.Extensions().ReplicaSets(namespace).Create(replicaset)
	return err
}

func deleteReplicaset(watcher *controller.Controller, name string) {
	if err := watcher.Clientset.Extensions().ReplicaSets(namespace).Delete(name, &api.DeleteOptions{}); err != nil {
		log.Errorln(err)
	}
}

func createDeployment(watcher *controller.Controller, name string, restikName string) error {
	deployment := &extensions.Deployment{
		ObjectMeta: api.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				controller.BackupConfig: restikName,
			},
		},
		Spec: extensions.DeploymentSpec{
			Replicas: 1,
			Selector: &unversioned.LabelSelector{
				MatchLabels: map[string]string{
					"app": "nginx",
				},
			},
			Template: *podTemplate,
		},
	}
	_, err := watcher.Clientset.Extensions().Deployments(namespace).Create(deployment)
	return err
}

func deleteDeployment(watcher *controller.Controller, name string) {
	if err := watcher.Clientset.Extensions().Deployments(namespace).Delete(name, &api.DeleteOptions{}); err != nil {
		log.Errorln(err)
	}
}

func createDaemonsets(watcher *controller.Controller, name string, backupName string) error {
	daemonset := &extensions.DaemonSet{
		ObjectMeta: api.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				controller.BackupConfig: backupName,
			},
		},
		Spec: extensions.DaemonSetSpec{
			Template: *podTemplate,
		},
	}
	_, err := watcher.Clientset.Extensions().DaemonSets(namespace).Create(daemonset)
	return err
}

func deleteDaemonset(watcher *controller.Controller, name string) {
	if err := watcher.Clientset.Extensions().DaemonSets(namespace).Delete(name, &api.DeleteOptions{}); err != nil {
		log.Errorln(err)
	}
}

func createStatefulSet(watcher *controller.Controller, name string, restikName string, svc string) error {
	s := &apps.StatefulSet{
		ObjectMeta: api.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				controller.BackupConfig: restikName,
			},
		},
		Spec: apps.StatefulSetSpec{
			Replicas:    1,
			Template:    *podTemplate,
			ServiceName: svc,
		},
	}
	container := api.Container{
		Name:            controller.ContainerName,
		Image:           image,
		ImagePullPolicy: api.PullAlways,
		Env: []api.EnvVar{
			{
				Name:  controller.RestikNamespace,
				Value: namespace,
			},
			{
				Name:  controller.RestikResourceName,
				Value: restikName,
			},
		},
	}
	container.Args = append(container.Args, "watch")
	container.Args = append(container.Args, "--v=10")
	backupVolumeMount := api.VolumeMount{
		Name:      "test-volume",
		MountPath: "/source_path",
	}
	sourceVolumeMount := api.VolumeMount{
		Name:      "restik-vol",
		MountPath: "/repo_path",
	}
	container.VolumeMounts = append(container.VolumeMounts, backupVolumeMount)
	container.VolumeMounts = append(container.VolumeMounts, sourceVolumeMount)
	s.Spec.Template.Spec.Containers = append(s.Spec.Template.Spec.Containers, container)
	s.Spec.Template.Spec.Volumes = append(s.Spec.Template.Spec.Volumes, api.Volume{
		Name: "restik-vol",
		VolumeSource: api.VolumeSource{
			EmptyDir: &api.EmptyDirVolumeSource{},
		},
	})
	_, err := watcher.Clientset.Apps().StatefulSets(namespace).Create(s)
	return err
}

func deleteStatefulset(watcher *controller.Controller, name string) {
	if err := watcher.Clientset.Apps().StatefulSets(namespace).Delete(name, &api.DeleteOptions{}); err != nil {
		log.Errorln(err)
	}
}

func createService(watcher *controller.Controller, name string) error {
	svc := &api.Service{
		ObjectMeta: api.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app": "nginx",
			},
		},
		Spec: api.ServiceSpec{
			Selector: map[string]string{
				"app": "nginx",
			},
			Ports: []api.ServicePort{
				{
					Port: 80,
					Name: "web",
				},
			},
		},
	}
	_, err := watcher.Clientset.Core().Services(namespace).Create(svc)
	return err
}

func deleteService(watcher *controller.Controller, name string) {
	err := watcher.Clientset.Core().Services(namespace).Delete(name, &api.DeleteOptions{})
	if err != nil {
		log.Errorln(err)
	}
}