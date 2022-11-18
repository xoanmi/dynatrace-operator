package manager

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/src/logger"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/config/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type ManagerStub struct {
	manager.Manager
}

<<<<<<< Updated upstream
// cluster.Cluster portion of manager.Manager interface
func (mgr *TestManager) GetClient() client.Client {
=======
func (mgr *ManagerStub) GetClient() client.Client {
>>>>>>> Stashed changes
	return struct{ client.Client }{}
}

func (mgr *ManagerStub) GetAPIReader() client.Reader {
	return struct{ client.Reader }{}
}

<<<<<<< Updated upstream
func (mgr *TestManager) GetScheme() *runtime.Scheme {
	return scheme.Scheme
}

func (mgr *TestManager) SetFields(interface{}) error {
	return nil
}

// manager.Manager interface
func (mgr *TestManager) GetControllerOptions() v1alpha1.ControllerConfigurationSpec {
	return v1alpha1.ControllerConfigurationSpec{}
}

func (mgr *TestManager) GetLogger() logr.Logger {
	return logger.Factory.GetLogger("test-manager")
}

func (mgr *TestManager) Add(manager.Runnable) error {
=======
func (mgr *ManagerStub) GetControllerOptions() v1alpha1.ControllerConfigurationSpec {
	return v1alpha1.ControllerConfigurationSpec{}
}

func (mgr *ManagerStub) GetScheme() *runtime.Scheme {
	return scheme.Scheme
}

func (mgr *ManagerStub) GetLogger() logr.Logger {
	return logger.Factory.GetLogger("test-manager")
}

func (mgr *ManagerStub) SetFields(interface{}) error {
	return nil
}

func (mgr *ManagerStub) Add(manager.Runnable) error {
>>>>>>> Stashed changes
	return nil
}

func (mgr *ManagerStub) Start(_ context.Context) error {
	return nil
}
