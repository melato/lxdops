package lxdops

// Config - Instance configuration
// An instance is a name that is used to launch a container or create LXD disk devices, typically both.
// The instance name, is the same as the base name of the config file, without the extension.
// It can be overridden by the -name flag
// Configuration sections are applied in the order that they are mentioned in the Config:
// - PreScripts
// - Packages
// - Users
// - Files
// - Scripts
// - Passwords
type Config struct {
	OS *OS
	// Description is
	Description string `yaml:"description,omitempty"`

	// Includes is a list of other configs that are to be included.
	// Include paths are either absolute or relative to the path of the including config.
	Includes []string `yaml:"include,omitempty"`

	// These are files or directories that must exist on the host.
	RequiredFiles []string `yaml:"require,omitempty"`

	// Origin is the name of a container and a snapshot to clone from.
	// It should have the form {container}/{snapshot}
	Origin string `yaml:"origin,omitempty"`

	// DeviceTemplate is the name of an instance, whose devices are copied (using rsync)
	// to a new instance with launch.
	// It should just have the form of (instance) where (instance) is an instance name
	DeviceTemplate string `yaml:"device-template,omitempty"`

	// DeviceOrigin is the name an "instance" to clone its zfs devices from
	// Each device zfs filesystem is cloned from @{snapshot}
	// When basing an instance on a template, it is preferably to specify a DeviceTemplate,
	// so that the files are copied, rather than cloned, so the container's disk devices
	// are not tied to the template.
	// DeviceOrigin is useful when cloning an instance, for testing or other purposes.
	DeviceOrigin string `yaml:"device-origin,omitempty"`
	// Filesystems are zfs filesystems or plain directories that are created
	// when an instance is created.  Devices are created inside filesystems.
	Filesystems []*Filesystem `yaml:"filesystems"`
	// Devices are disk devices that are directories within the instance filesystems
	// They are created and attached to the container via the instance profile
	Devices []*Device `yaml:"devices,omitempty"`
	// Profiles are attached to the container.  The instance profile should not be listed here.
	Profiles []string `yaml:"profiles,omitempty"`

	// PreScripts are scripts that are executed early, before packages, users, files, or Scripts
	PreScripts []*Script `yaml:"pre-scripts,omitempty"`

	// Packages are OS packages that are installed when the instance is launched
	Packages []string `yaml:"packages,omitempty"`

	// Users are OS users that are created when the instance is launched
	Users []*User `yaml:"users,omitempty"`
	// Files are files that are copied from the host to the container when the instance is launched (as with lxc file push).
	Files []*File `yaml:"files,omitempty"`
	// Scripts are scripts that are executed in the container (as with lxc exec)
	Scripts []*Script `yaml:"scripts,omitempty"`
	// Passwords are a list of OS accounts, whose password is set to a random password
	Passwords []string `yaml:"passwords,omitempty"`
	// Stop specifies that the container should be stopped at the end of the configuration
	Stop bool `yaml:"stop,omitempty"`
	// Snapshot specifies that that the container should be snapshoted with this name at the end of the configuration process.
	Snapshot string `yaml:"snapshot,omitempty"`

	// Properties are used for token substitution where patterns are used (e.g. in filesystem paths)
	Properties map[string]string `yaml:"properties,omitempty"`
	// ProfilePattern specifies how the instance profile should be named.
	// It defaults to "(container).lxdops", where (container) is the name of the instance
	ProfilePattern string `yaml:"profile-pattern"`
}

// OS specifies the container OS
type OS struct {
	// Name if the name of the container image, without the version number.
	// All included configuration files should have the same OS Name.
	// Supported OS names are "alpine", "debian", "ubuntu".
	// Support for an OS is the ability to determine the LXD image, install packages, create users, set passwords
	Name string `yaml:"name,omitempty" xml:"name,attr,omitempty"`
	// Version is the image version, e.g. 3.13, 10.04.  The image name is composed of Name/Version
	// Version is optional in configuration files, but the final assembled configuration file should have a OS Version.
	// It should typically be specified in one configuration file that is included by all other configuration files that use use this OS
	Version string `yaml:"version,omitempty" xml:"version,attr,omitempty"`
	osType  OSType `xml:"-"`
}

// Filesystem is a ZFS filesystem or a plain directory that is created when an instance is created
// The disk devices of an instance are created as subdirectories of a Filesystem
type Filesystem struct {
	// Id is the identifier used to reference the filesystm by devices.
	// An empty Id is a valid identifier, which can typically be used to denote a default filesystem
	Id string `xml:"name,attr"`
	// Pattern is a pattern that is used to produce the directory or zfs filesystem
	// If the pattern begins with '/', it is a directory
	// If it does not begin with '/', it is a zfs filesystem name
	Pattern string `xml:"name,attr"`
	// Zfsproperties is a list of properties that are set when a zfs filesystem is created or cloned
	Zfsproperties map[string]string `yaml:",omitempty"`
}

// A Device is an LXD disk device that is attached to the instance profile, which in turn is attached to a container
type Device struct {
	// Path is the device "path" in the LXD disk device
	Path string `xml:"path,attr"`
	// Name is the name of the LXD disk device
	Name string `xml:"name,attr"`

	// Filesystem is the Filesystem Id that this device belongs to
	Filesystem string `yaml:",omitempty"`

	// Dir is the subdirectory of the Device, relative to its Filesystem
	// If empty, it default to the device Name
	// If Dir == ".", the device source is the same as the Filesystem directory
	// Rarely used:
	// Dir goes through pattern substitution, using parenthesized tokens, for example (container)
	// Dir may be absolute, but this is no longer necessary now that filesystems are specified, since one can define the "/" filesystem.
	Dir string `yaml:",omitempty"`
}

// File specifies a file that is copied from the host to the container
type File struct {
	// Path is the file path in the container
	Path string

	// Source is the file path on the host
	Source string

	// Recursive is not supported and will be removed.  Only single files are supported.
	Recursive bool

	// The file's mode as a string, e.g. 0775
	Mode string

	// Uid is the file's numeric uid in the container
	Uid int

	// Gid is the file's numeric gid in the container
	Gid int

	// User is the file's owner name in the container.  It is an error if both uid and user are set.
	User string

	// Group is the file's group owner name in the container.  It is an error if both gid and group are set.
	Group string
}

// Script specifies a sh script that is run in the container
type Script struct {
	// An optional name that identifies the script, useful for debugging/testing
	Name string `yaml:"name"`

	// File is an optional host file that contains the script content.
	// It should be an executable.  It is copied to the container in /root/ and run there.
	File string `yaml:"file,omitempty"`

	// Reboot specifies that the container should be rebooted after running this script
	// This may be needed when replacing /etc files
	// Reboot may be slow, so avoid it, if possible
	Reboot bool `yaml:"reboot,omitempty"`

	// Body is the content of the script
	// It is passed as the stdin to sh
	Body string `yaml:"body,omitempty"`

	// Dir is the directory in the container to set as the working directory when running the script
	Dir string `yaml:"dir,omitempty"`

	// Uid is the container uid to run the script as
	Uid uint32 `yaml:"uid,omitempty"`

	// Gid is the container gid to run the script as
	Gid uint32 `yaml:"gid,omitempty"`
}

// An OS user
type User struct {
	// Name is the user name.  If missing, the user takes the name of current host user
	Name string `yaml:"name"`
	// Uid is an optional uid for the user
	Uid string `yaml:"uid,omitempty"`
	// Sudo gives full passwordless sudo privileges to the user
	Sudo bool `yaml:"sudo,omitempty"`
	// Ssh specifies that the current user's ~.ssh/authorized_keys should be copied from the host to this user
	Ssh bool `yaml:"ssh,omitempty"`
	// Shell is the user shell
	Shell string `yaml:"shell,omitempty"`
	// Home is the user home directory, optional
	Home string `yaml:"home,omitempty"`
	// Groups is a list of groups that the user is added to
	Groups []string `yaml:"groups,omitempty"`
}
