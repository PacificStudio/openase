package hradvisor

type recommendationSupportStatus string

const (
	recommendationSupportSupportedNow          recommendationSupportStatus = "supported_now"
	recommendationSupportIntentionallyDisabled recommendationSupportStatus = "intentionally_unsupported"
	recommendationSupportPlanned               recommendationSupportStatus = "planned_not_yet_implemented"
)

type roleRecommendationSupport struct {
	RoleSlug string
	Status   recommendationSupportStatus
	Reason   string
}

var roleRecommendationSupportMatrix = map[string]roleRecommendationSupport{
	"dispatcher": {
		RoleSlug: "dispatcher",
		Status:   recommendationSupportSupportedNow,
		Reason:   "Backlog pressure is observable from ticket stages and workflow bindings.",
	},
	"harness-optimizer": {
		RoleSlug: "harness-optimizer",
		Status:   recommendationSupportSupportedNow,
		Reason:   "Workflow retries, paused tickets, and failure bursts expose harness quality drift directly.",
	},
	"env-provisioner": {
		RoleSlug: "env-provisioner",
		Status:   recommendationSupportSupportedNow,
		Reason:   "Environment repair demand can be inferred from explicit repair lanes and repeated stalled retries.",
	},
	"fullstack-developer": {
		RoleSlug: "fullstack-developer",
		Status:   recommendationSupportSupportedNow,
		Reason:   "Generic implementation demand is observable from coding tickets and in-progress projects without agent coverage.",
	},
	"frontend-engineer": {
		RoleSlug: "frontend-engineer",
		Status:   recommendationSupportSupportedNow,
		Reason:   "Dedicated frontend lanes can be inferred from workflow-bound ticket lanes.",
	},
	"backend-engineer": {
		RoleSlug: "backend-engineer",
		Status:   recommendationSupportSupportedNow,
		Reason:   "Dedicated backend and API lanes can be inferred from workflow-bound ticket lanes.",
	},
	"qa-engineer": {
		RoleSlug: "qa-engineer",
		Status:   recommendationSupportSupportedNow,
		Reason:   "Testing demand is visible through coding load, test lanes, and workflow bindings.",
	},
	"devops-engineer": {
		RoleSlug: "devops-engineer",
		Status:   recommendationSupportSupportedNow,
		Reason:   "Deploy and release lanes are observable from ticket queues and workflow bindings.",
	},
	"security-engineer": {
		RoleSlug: "security-engineer",
		Status:   recommendationSupportSupportedNow,
		Reason:   "Security demand is visible from security lanes, failure bursts, and error pressure.",
	},
	"technical-writer": {
		RoleSlug: "technical-writer",
		Status:   recommendationSupportSupportedNow,
		Reason:   "Documentation demand is visible from doc lanes and documentation drift activity.",
	},
	"code-reviewer": {
		RoleSlug: "code-reviewer",
		Status:   recommendationSupportSupportedNow,
		Reason:   "Review bottlenecks are observable from review lanes that lack pickup coverage.",
	},
	"product-manager": {
		RoleSlug: "product-manager",
		Status:   recommendationSupportSupportedNow,
		Reason:   "Pre-execution planning gaps are visible from empty planned projects.",
	},
	"market-analyst": {
		RoleSlug: "market-analyst",
		Status:   recommendationSupportIntentionallyDisabled,
		Reason:   "Market research requires external demand signals; HR Adviser must not guess that from internal project traffic alone.",
	},
	"research-ideation": {
		RoleSlug: "research-ideation",
		Status:   recommendationSupportSupportedNow,
		Reason:   "Research discovery demand can be inferred from project intent and research-oriented lanes.",
	},
	"experiment-runner": {
		RoleSlug: "experiment-runner",
		Status:   recommendationSupportSupportedNow,
		Reason:   "Experiment execution lanes are observable from ticket queues and workflow bindings.",
	},
	"report-writer": {
		RoleSlug: "report-writer",
		Status:   recommendationSupportSupportedNow,
		Reason:   "Research writing and reporting lanes are observable from ticket queues and workflow bindings.",
	},
	"data-analyst": {
		RoleSlug: "data-analyst",
		Status:   recommendationSupportPlanned,
		Reason:   "Auto-recommending data analysis needs dataset and measurement signals that the current snapshot does not include yet.",
	},
}

func recommendationSupport(roleSlug string) (roleRecommendationSupport, bool) {
	support, ok := roleRecommendationSupportMatrix[roleSlug]
	return support, ok
}
