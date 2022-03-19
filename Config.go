package lxdops

// HostPath is a file path on the host, which is either absolute or relative to a base directory
// When a config file includes another config file, the base directory is the directory of the including file
type HostPath string

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
	// ConfigTop fields are not merged with included files
	ConfigTop `yaml:",inline"`
	// ConfigInherit fields are merged with all included files, depth first
	ConfigInherit `yaml:",inline"`
}

type ConfigTop struct {
	// Description is provided for documentation
	Description string `yaml:"description"`

	// Stop specifies that the container should be stopped at the end of the configuration
	Stop bool `yaml:"stop"`

	// Snapshot specifies that that the container should be snapshoted with this name at the end of the configuration process.
	Snapshot string `yaml:"snapshot"`
}

type ConfigInherit struct {
	// Properties provide key-value pairs used for pattern substitution.
	// They override built-in properties
	// Properties from all included files are merged before they are applied.  Properties cannot override non-empty properties,
	// in order to avoid unexpected behavior that depends on the order of included files.
	Properties map[string]string `yaml:"properties"`

	OS *OS

	// Project is the LXD project where the container is
	Project string `yaml:"project"`

	// Experimental: The name of the container. Defaults to (instance)
	Container Pattern `yaml:"container,omitempty"`

	// ProfilePattern specifies how the instance profile should be named.
	// It defaults to "(instance).lxdops"
	Profile Pattern `yaml:"profile-pattern"`

	// ProfileConfig specifies Config entries to be added to the instance profile.
	// This was meant for creating templates with boot.autostart: "false",
	// without needing to use profiles external to lxdops.
	ProfileConfig map[string]string `yaml:"profile-config"`

	// Source specifies where to copy or clone the instance from
	Source `yaml:",inline"`

	// Extra options passed to lxc launch.
	LxcOptions []string `yaml:"lxc-options,omitempty,flow"`

	// Include is a list of other configs that are to be included.
	// Include paths are either absolute or relative to the path of the including config.
	Include []HostPath `yaml:"include,omitempty"`

	// Filesystems are zfs filesystems or plain directories that are created
	// when an instance is created.  Devices are created inside filesystems.
	Filesystems map[string]*Filesystem `yaml:"filesystems"`
	// Devices are disk devices that are directories within the instance filesystems
	// They are created and attached to the container via the instance profile
	Devices map[string]*Device `yaml:"devices"`
	// Profiles are attached to the container.  The instance profile should not be listed here.

	// The owner (uid:gid) for new devices
	DeviceOwner Pattern `yaml:"device-owner"`

	Profiles []string `yaml:"profiles"`

	// PreScripts are scripts that are executed early, before packages, users, files, or Scripts
	PreScripts []*Script `yaml:"pre-scripts"`

	// Packages are OS packages that are installed when the instance is launched
	Packages []string `yaml:"packages"`

	// Users are OS users that are created when the instance is launched
	Users []*User `yaml:"users"`
	// Files are files that are copied from the host to the container when the instance is launched (as with lxc file push).
	Files []*File `yaml:"files"`
	// Scripts are scripts that are executed in the container (as with lxc exec)
	Scripts []*Script `yaml:"scripts"`
	// Passwords are a list of OS accounts, whose password is set to a random password
	Passwords []string `yaml:"passwords"`
	// Pull files are pulled from the container to the host (as with lxc file pull).
	PullFiles []*PullFile `yaml:"pull"`
}

// Source specifies how to copy or clone the instance container, filesystem, and device directories.
// When DeviceTemplate is specified, the filesystems are copied with rsync.
// When DeviceOrigin is specified, the filesystems are cloned with zfs-clone
// The filesystems that are copied are determined by applying the source instance name to the filesystems of this config,
// or to the filesystems of a source config.
//
// When basing an instance on a template with few skeleton files, it is preferable to copy with a DeviceTemplate,
// so the container's disk devices are not tied to the template.
//
// Example:
// suppose test-a.yaml has:
//   origin: a/copy
//   filesystems: "default": "z/test/(instance)"
//   device-origin: a@copy
//   source-filesystems "default": "z/prod/(instance)"
//   devices: home, path=/home, filesystem=default
// This would do something like:
//    zfs clone z/prod/a@copy z/test/test-a
//    lxc copy --container-only a/copy test-a
//    lxc profile create test-a.lxdops
//    lxc profile device add test-a.lxdops home disk path=/home source=/z/test/test-a/home
//    lxc profile add test-a test-a.lxdops
type Source struct {
	// origin is the name of a container and a snapshot to clone from.
	// It has the form [<project>_]<container>[/<snapshot>]
	// It overrides SourceConfig
	Origin Pattern `yaml:"origin"`

	// device-template is the name of an instance, whose devices are copied (using rsync)
	// to a new instance with launch.
	// The devices are copied from the filesystems specified in SourceConfig, or this config.
	DeviceTemplate Pattern `yaml:"device-template"`

	// device-origin is the name an instance and a short snapshot name.
	// It has the form <instance>@<snapshot> where <instance> is an instance name,
	// and @<snapshot> is a the short snapshot name of the instance filesystems.
	// Each device zfs filesystem is cloned from @<snapshot>
	// The filesytems are those specified in SourceConfig, if any, otherwise this config.
	DeviceOrigin Pattern `yaml:"device-origin"`

	// Experimental: source-config specifies a config file that is used to determine:
	//   - The LXD project, container, and snapshot to clone when launching the instance.
	//   - The source filesystems used for cloning filesystems or copying device directories.
	// The name of the instance used for the source filesystems
	// is the base name of the filename, without the extension.
	// Various parts of these items can be overriden by other source properties above
	SourceConfig HostPath `yaml:"source-config,omitempty"`
}

// OS specifies the container OS
type OS struct {
	// Name if the name of the container image, without the version number.
	// All included configuration files should have the same OS Name.
	// Supported OS names are "alpine", "debian", "ubuntu".
	// Support for an OS is the ability to determine the LXD image, install packages, create users, set passwords
	Name string `yaml:"name"`

	// Image is the image name (with optional remote), used when launching a container.
	// If Image is missing, an image name is constructed from Version.
	Image Pattern `yaml:"image"`

	// Version is used if Image is not specified.  The image name is composed of Name/Version
	// Version is optional in configuration files, but the final assembled configuration file should have a OS Version.
	Version Pattern `yaml:"version"`

	osType OSType
}

// Filesystem is a ZFS filesystem or a plain directory that is created when an instance is created
// The disk devices of an instance are created as subdirectories of a Filesystem
type Filesystem struct {
	// Pattern is a pattern that is used to produce the directory or zfs filesystem
	// If the pattern begins with '/', it is a directory
	// If it does not begin with '/', it is a zfs filesystem name
	Pattern Pattern
	// Zfsproperties is a list of properties that are set when a zfs filesystem is created or cloned
	Zfsproperties map[string]string `yaml:""`
}

// A Device is an LXD disk device that is attached to the instance profile, which in turn is attached to a container
type Device struct {
	// Path is the device "path" in the LXD disk device
	Path string

	// Filesystem is the Filesystem Id that this device belongs to
	Filesystem string `yaml:"filesystem"`

	// Dir is the subdirectory of the Device, relative to its Filesystem
	// If empty, it default to the device Name
	// If Dir == ".", the device source is the same as the Filesystem directory
	// Rarely used:
	// Dir goes through pattern substitution, using parenthesized tokens, for example (instance)
	// Dir may be absolute, but this is no longer necessary now that filesystems are specified, since one can define the "/" filesystem.
	Dir Pattern `yaml:""`
}

// File specifies a file that is copied from the host to the container
type File struct {
	// Path is the file path in the container
	Path string

	// Source is the file path on the host
	Source HostPath

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

// PullFile specifies a file that is copied from the host to the container
type PullFile struct {
	// Path is the file path in the container
	Path string

	// Target is the file path on the host
	Target Pattern `yaml:"target"`

	// The file's mode as an octal string, e.g. 0644
	Mode string
}

// Script specifies a sh script that is run in the container
type Script struct {
	// An optional name that identifies the script, useful for debugging/testing
	Name string `yaml:"name"`

	// File is an optional host file that contains the script content.
	// It should be an executable.  It is copied to the container in /root/ and run there.
	File HostPath `yaml:"file"`

	// Reboot specifies that the container should be rebooted after running this script
	// This may be needed when replacing /etc files
	// Reboot may be slow, so avoid it, if possible
	Reboot bool `yaml:"reboot"`

	// Body is the content of the script
	// It is passed as the stdin to sh
	Body string `yaml:"body"`

	// Dir is the directory in the container to set as the working directory when running the script
	Dir string `yaml:"dir"`

	// Uid is the container uid to run the script as
	Uid uint32 `yaml:"uid"`

	// Gid is the container gid to run the script as
	Gid uint32 `yaml:"gid"`
}

// An OS user
type User struct {
	// Name is the user name.  If missing, the user takes the name of current host user
	Name string `yaml:"name"`
	// Uid is an optional uid for the user
	Uid string `yaml:"uid"`
	// Sudo gives full passwordless sudo privileges to the user
	Sudo bool `yaml:"sudo"`
	// Ssh specifies that the current user's ~.ssh/authorized_keys should be copied from the host to this user
	Ssh bool `yaml:"ssh"`
	// Shell is the user shell
	Shell string `yaml:"shell"`
	// Home is the user home directory, optional
	Home string `yaml:"home"`
	// Groups is a list of groups that the user is added to
	Groups []string `yaml:"groups"`
}
