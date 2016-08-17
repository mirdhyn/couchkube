package main

import (
	"fmt"
	"log"

	"k8s.io/kubernetes/pkg/api"
	vapi "k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/apis/extensions"
)

func CreateDeployment(db Database) (err error) {
	name := *db.Name

	deploySpec := &extensions.Deployment{
		TypeMeta: vapi.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "extensions/v1beta1",
		},
		ObjectMeta: api.ObjectMeta{
			Name: name,
		},
		Spec: extensions.DeploymentSpec{
			Replicas: 1,
			Template: api.PodTemplateSpec{
				ObjectMeta: api.ObjectMeta{
					Name:   name,
					Labels: map[string]string{"database": name},
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						api.Container{
							Name:  "couchdb",
							Image: "klaemo/couchdb:latest",
							Ports: []api.ContainerPort{
								api.ContainerPort{ContainerPort: 5984, Protocol: api.ProtocolTCP},
							},
							ImagePullPolicy: api.PullIfNotPresent,
							Env: []api.EnvVar{
								api.EnvVar{
									Name:  "COUCHDB_USER",
									Value: couchdbUser,
								},
								api.EnvVar{
									Name:  "COUCHDB_PASSWORD",
									Value: couchdbPassword,
								},
							},
						},
					},
				},
			},
		},
	}

	deploy := kube.Extensions().Deployments(namespace)
	_, err = deploy.Create(deploySpec)
	return err

}

func CreateService(db Database) (err error) {
	name := *db.Name
	// Define service spec.
	serviceSpec := &api.Service{
		TypeMeta: vapi.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: api.ObjectMeta{
			Name: name,
		},
		Spec: api.ServiceSpec{
			Type:     api.ServiceTypeLoadBalancer,
			Selector: map[string]string{"database": name},
			Ports: []api.ServicePort{
				api.ServicePort{
					Protocol: api.ProtocolTCP,
					Port:     5984,
				},
			},
		},
	}

	_, err = kube.Services(namespace).Create(serviceSpec)
	if err != nil {
		return fmt.Errorf("failed to create service: %s", err)
	}
	log.Println("service created")

	GetService(name)
	// get service - wait for endpoint to be created
	// create db
	// setup replication

	return nil
}

func GetService(name string) (string, error) {
	s, err := kube.Services(namespace).Get(name)
	if err != nil {
		log.Fatalln("Can't get service:", err)
		return "", err
	}

	var endpoint string
	for ingress := s.Status.LoadBalancer.Ingress; len(ingress) > 0; {

		ip := ingress[0].IP
		hostname := ingress[0].Hostname
		port := s.Spec.Ports[0].Port

		switch {
		case len(ip) != 0:
			endpoint = fmt.Sprintf("http://%s:%d", ip, port)
		case len(hostname) != 0:
			endpoint = fmt.Sprintf("http://%s:%d", hostname, port)
		default:
			return "", fmt.Errorf("failed to find service")
		}
	}

	return endpoint, nil
}
