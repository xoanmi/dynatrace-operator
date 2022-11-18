package manager

import (
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

//go:generate mockery --config ../../../mockery.yaml --name=Provider
type Provider interface {
	CreateManager(namespace string, config *rest.Config) (manager.Manager, error)
}

<<<<<<< Updated upstream
type Manager interface {
	manager.Manager
}
=======
//go:generate mockery --config ../../../mockery.yaml --srcpkg=sigs.k8s.io/controller-runtime/pkg/manager --name=Manager

// go:generate docker run --mount type=bind,src=$PWD/../../..,dst=/src -w /src vektra/mockery --srcpkg=sigs.k8s.io/controller-runtime/pkg/manager --output $PWD/mocks --with-expecter --testonly --case snake --name=Manager
>>>>>>> Stashed changes
