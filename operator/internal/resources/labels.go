package resources

// Common label keys used across all resources.
const (
	LabelUserID    = "etalbaas.io/user-id"
	LabelProjectID = "etalbaas.io/project-id"
	LabelPlan      = "etalbaas.io/plan"
	LabelManagedBy = "app.kubernetes.io/managed-by"
	LabelComponent = "app.kubernetes.io/component"
	LabelPartOf    = "app.kubernetes.io/part-of"

	ManagedByValue = "etalbaas-operator"
	PartOfValue    = "etalbaas"
)

// CommonLabels returns the standard labels for all resources in a project.
func CommonLabels(projectID, userID, plan string) map[string]string {
	return map[string]string{
		LabelProjectID: projectID,
		LabelUserID:    userID,
		LabelPlan:      plan,
		LabelManagedBy: ManagedByValue,
		LabelPartOf:    PartOfValue,
	}
}

// ComponentLabels returns labels for a specific component within a project.
func ComponentLabels(projectID, userID, plan, component string) map[string]string {
	labels := CommonLabels(projectID, userID, plan)
	labels[LabelComponent] = component
	return labels
}
