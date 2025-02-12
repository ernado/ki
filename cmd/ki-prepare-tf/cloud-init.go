package main

type CloudConfig struct {
	Packages   []string `yaml:"packages"`
	Users      []User   `yaml:"users"`
	WriteFiles []File   `yaml:"write_files"`
	RunCmd     []string `yaml:"runcmd"`
}

type User struct {
	Name              string   `yaml:"name"`
	SSHAuthorizedKeys []string `yaml:"ssh-authorized-keys"`
	Sudo              string   `yaml:"sudo"`
	Shell             string   `yaml:"shell"`
}

type File struct {
	Path        string `yaml:"path"`
	Content     string `yaml:"content"`
	Permissions string `yaml:"permissions"`
}
