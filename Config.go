package lxdops

/** Config - Container configuration
Configuration sections are applied in this order:
- Scripts with First = true
- Packages
- Users
- Files
- Scripts with First = false
- Passwords
*/
type Config struct {
	OS          *OS
	Description string `yaml:"description,omitempty"`
	/** Include other configs */
	Includes []string `yaml:"include,omitempty"`
	/** Files or directories that must exist on the host */
	RequiredFiles []string `yaml:"require,omitempty"`
	HostFS        string   `yaml:"host-fs,omitempty"`

	Origin         string        `yaml:"origin,omitempty"`
	DeviceTemplate string        `yaml:"device-template,omitempty"`
	DeviceOrigin   string        `yaml:"device-origin,omitempty"`
	Filesystems    []*Filesystem `yaml:"filesystems"`
	Devices        []*Device     `yaml:"devices,omitempty"`
	Profiles       []string      `yaml:"profiles,omitempty"`

	PreScripts []*Script `yaml:"pre-scripts,omitempty"`

	Packages  []string  `yaml:"packages,omitempty"`
	Users     []*User   `yaml:"users,omitempty"`
	Files     []*File   `yaml:"files,omitempty"`
	Scripts   []*Script `yaml:"scripts,omitempty"`
	Passwords []string  `yaml:"passwords,omitempty"`
	Snapshot  string    `yaml:"snapshot,omitempty"`
	Stop      bool      `yaml:"stop,omitempty"`

	Properties     map[string]string `yaml:"properties,omitempty"`
	ProfilePattern string            `yaml:"profile-pattern"`
}

type OS struct {
	Name    string `yaml:"name,omitempty" xml:"name,attr,omitempty"`
	Version string `yaml:"version,omitempty" xml:"version,attr,omitempty"`
	osType  OSType `xml:"-"`
}

type Filesystem struct {
	/** Filesystem identifier, referenced by devices.  Can be empty to denote a default filesystem */
	Id string `xml:"name,attr"`
	/** A directory or zfs dataset pattern */
	Pattern       string            `xml:"name,attr"`
	Zfsproperties map[string]string `yaml:",omitempty"`
}

type Device struct {
	/** The device path in the container */
	Path string `xml:"path,attr"`
	/** The name of the device in the profile. */
	Name string `xml:"name,attr"`

	/** Filesystem.Id */
	Filesystem string `yaml:",omitempty"`

	/** A (sub) directory pattern (optional).
	If Dir begins with "/", use (Dir)
	If Dir is empty, use (Filesystem.Dir)/(Name)
	If Dir == ".", use (Filesystem.Dir)
	Otherwise, use /(Filesystem.Dir)/(Dir)
	Use pattern substitution on (Dir)
	*/
	Dir string `yaml:",omitempty"`
}

type File struct {
	/** The destination path.
	 */
	Path string

	/** The source path.
	 */
	Source string

	Recursive bool

	Mode string

	Uid int

	Gid int

	User string

	Group string
}

type Script struct {
	/** A name that identifies the script. */
	Name string `xml:"name,attr" yaml:"name"`

	/** The file to run.  This is a file on the host, that is copied to the container in /root/ and run there.
	 */
	File string `xml:"file,attr" yaml:"file,omitempty"`

	/** Reboot after running this script */
	Reboot bool `xml:"reboot,attr" yaml:"reboot,omitempty"`

	/** The content of the script. */
	Body string `xml:",cdata" yaml:"body,omitempty"`

	/** The directory to run the script in. */
	Dir string `xml:"dir,attr" yaml:"dir,omitempty"`

	/** The uid to run as */
	Uid uint32 `xml:"uid,attr" yaml:"uid,omitempty"`

	/** The gid to run as */
	Gid uint32 `xml:"gid,attr" yaml:"gid,omitempty"`
}

type User struct {
	/** Current - Use the name and uid of the user that is running this program */
	//Current bool     `xml:"current,attr" yaml:"current"`
	Name   string   `xml:"name,attr" yaml:"name"`
	Uid    string   `xml:"uid,attr" yaml:"uid,omitempty"`
	Sudo   bool     `xml:"sudo,attr" yaml:"sudo,omitempty"`
	Ssh    bool     `xml:"ssh,attr" yaml:"ssh,omitempty"`
	Shell  string   `xml:"shell,attr" yaml:"shell,omitempty"`
	Home   string   `xml:"home,attr" yaml:"home,omitempty"`
	Groups []string `xml:"group" yaml:"groups,omitempty"`
}
