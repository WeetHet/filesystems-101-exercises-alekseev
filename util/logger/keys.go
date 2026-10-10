package logger

// Standard field keys.
const (
	//KeyMessage identifis a log message itself.
	KeyMessage = "msg"
	// KeyLevel identifies the log level of the message.
	KeyLevel = "level"

	// KeyService identifies a service by its name.
	KeyService = "service"
	// KeyVersion contains service's version information.
	KeyVersion = "version"

	// KeySubsystem identifies a particular subsystem, such as gsuite or
	// archive.
	KeySubsystem = "subsys"
	// KeyComponent identifies a particular component within a subsystem,
	// such as contacts or io.
	KeyComponent = "component"
	// KeyProcess names an ongoing process within a component or service,
	// such as backup or restore.
	KeyProcess = "process"

	// KeyOrg is organization ID
	KeyOrg = "org"
	// KeyTenant describes a tenant on behalf of which the request is made.
	KeyTenant = "tenant"
	// KeyResource is the ID of the resource.
	KeyResource = "resource"
	// KeyResourceName is the name of the resource.
	KeyResourceName = "resource_name"
	// KeyTopic is a message queue topic name
	KeyTopic = "topic"
	// KeySession is a session name or id
	KeySession = "session"
	// KeyOwner is a resource owner id (usually email)
	KeyOwner = "owner"
	// KeyAccount is an arbitrary account name (usually email)
	KeyAccount = "account"
	// KeyJob is an arbitrary job id
	KeyJob = "job_id"
	// KeyTaskID is an arbitrary task id
	KeyTask = "task_id"

	// KeyError describes an error.
	KeyError = "error"
)
