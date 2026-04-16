package prometheus

import "fmt"

// BuildCPUUtilizationQuery returns a PromQL query for CPU utilization
// showing the hottest pod per deployment (max across pods).
func BuildCPUUtilizationQuery(namespacesRegex string) string {
	return fmt.Sprintf(`max by (deployment, namespace) (
    sum by (pod, deployment, namespace) (
        label_replace(
            rate(container_cpu_usage_seconds_total{namespace=~"%s", container!=""}[5m]),
            "deployment", "$1", "pod", "(.*)-[a-z0-9]+-[a-z0-9]+"
        )
    )
    /
    sum by (pod, deployment, namespace) (
        label_replace(
            kube_pod_container_resource_requests{namespace=~"%s", container!="", resource="cpu"},
            "deployment", "$1", "pod", "(.*)-[a-z0-9]+-[a-z0-9]+"
        )
    )
) * 100`, namespacesRegex, namespacesRegex)
}

// BuildRAMUtilizationQuery returns a PromQL query for RAM utilization
// showing the hottest pod per deployment (max across pods).
func BuildRAMUtilizationQuery(namespacesRegex string) string {
	return fmt.Sprintf(`max by (deployment, namespace) (
    sum by (pod, deployment, namespace) (
        label_replace(
            container_memory_working_set_bytes{namespace=~"%s", container!=""},
            "deployment", "$1", "pod", "(.*)-[a-z0-9]+-[a-z0-9]+"
        )
    )
    /
    sum by (pod, deployment, namespace) (
        label_replace(
            kube_pod_container_resource_requests{namespace=~"%s", container!="", resource="memory"},
            "deployment", "$1", "pod", "(.*)-[a-z0-9]+-[a-z0-9]+"
        )
    )
) * 100`, namespacesRegex, namespacesRegex)
}

// BuildDeploymentCPUQuery returns a PromQL query for CPU history of a specific deployment.
// Shows the hottest pod per deployment (max across pods).
func BuildDeploymentCPUQuery(deployment, namespace string) string {
	return fmt.Sprintf(`max by (deployment, namespace) (
    sum by (pod, deployment, namespace) (
        label_replace(
            rate(container_cpu_usage_seconds_total{namespace="%s", container!=""}[5m]),
            "deployment", "$1", "pod", "(.*)-[a-z0-9]+-[a-z0-9]+"
        )
    )
    /
    sum by (pod, deployment, namespace) (
        label_replace(
            kube_pod_container_resource_requests{namespace="%s", container!="", resource="cpu"},
            "deployment", "$1", "pod", "(.*)-[a-z0-9]+-[a-z0-9]+"
        )
    )
) * 100`, namespace, namespace)
}

// BuildDeploymentRAMQuery returns a PromQL query for RAM history of a specific deployment.
// Shows the hottest pod per deployment (max across pods).
func BuildDeploymentRAMQuery(deployment, namespace string) string {
	return fmt.Sprintf(`max by (deployment, namespace) (
    sum by (pod, deployment, namespace) (
        label_replace(
            container_memory_working_set_bytes{namespace="%s", container!=""},
            "deployment", "$1", "pod", "(.*)-[a-z0-9]+-[a-z0-9]+"
        )
    )
    /
    sum by (pod, deployment, namespace) (
        label_replace(
            kube_pod_container_resource_requests{namespace="%s", container!="", resource="memory"},
            "deployment", "$1", "pod", "(.*)-[a-z0-9]+-[a-z0-9]+"
        )
    )
) * 100`, namespace, namespace)
}
