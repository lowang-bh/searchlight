package icinga

import (
	api "github.com/appscode/searchlight/apis/monitoring/v1alpha1"
	core "k8s.io/api/core/v1"
)

type NodeHost struct {
	commonHost

	//*types.Context
}

func NewNodeHost(IcingaClient *Client, verbosity string) *NodeHost {
	return &NodeHost{
		commonHost: commonHost{
			IcingaClient: IcingaClient,
			verbosity:    verbosity,
		},
	}
}

func (h *NodeHost) getHost(namespace string, node *core.Node) IcingaHost {
	nodeIP := "127.0.0.1"
	for _, ip := range node.Status.Addresses {
		if ip.Type == internalIP {
			nodeIP = ip.Address
			break
		}
	}
	return IcingaHost{
		ObjectName:     node.Name,
		Type:           TypeNode,
		AlertNamespace: namespace,
		IP:             nodeIP,
	}
}

// set Alert in Icinga LocalHost
func (h *NodeHost) Apply(alert *api.NodeAlert, node *core.Node) error {
	alertSpec := alert.Spec
	kh := h.getHost(alert.Namespace, node)

	if err := h.reconcileIcingaHost(kh); err != nil {
		return err
	}

	has, err := h.checkIcingaService(alert.Name, kh)
	if err != nil {
		return err
	}

	if alertSpec.Paused {
		if has {
			if err := h.deleteIcingaService(alert.Name, kh); err != nil {
				return err
			}
		}
		return nil
	}

	attrs := make(map[string]interface{})
	if alertSpec.CheckInterval.Seconds() > 0 {
		attrs["check_interval"] = alertSpec.CheckInterval.Seconds()
	}
	for key, val := range alertSpec.Vars {
		attrs[IVar(key)] = val
	}

	if !has {
		attrs["check_command"] = alertSpec.Check
		if err := h.createIcingaService(alert.Name, kh, attrs); err != nil {
			return err
		}
	} else {
		if err := h.updateIcingaService(alert.Name, kh, attrs); err != nil {
			return err
		}
	}

	return h.reconcileIcingaNotification(alert, kh)
}

func (h *NodeHost) Delete(alertNamespace, alertName string, node *core.Node) error {
	kh := h.getHost(alertNamespace, node)

	if err := h.deleteIcingaService(alertName, kh); err != nil {
		return err
	}
	return h.deleteIcingaHost(kh)
}

func (h *NodeHost) DeleteChecks(cmd string) error {
	return h.deleteIcingaServiceForCheckCommand(cmd)
}
