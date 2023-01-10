package exoscale

const (
	// OrganizationLabel represents the label used for organization when fetching the metrics
	OrganizationLabel = "appuio.io/organization"
	// NamespaceLabel represents the label used for namespace when fetching the metrics
	NamespaceLabel = "crossplane.io/claim-namespace"

	// ExoscaleBillingHour represents the hour when metrics are collected
	ExoscaleBillingHour = 6
	// ExoscaleTimeZone represents the time zone for ExoscaleBillingHour
	ExoscaleTimeZone = "UTC"
)
