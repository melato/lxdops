module melato.org/lxdops

go 1.18

replace (
	melato.org/cloudconfig => ../cloudconfig
	melato.org/cloudconfiglxd => ../cloudconfig-lxd
	melato.org/yaml => ../yaml
)

require (
	github.com/lxc/lxd v0.0.0-20230203092445-70b38dec97c2
	melato.org/cloudconfig v0.0.0-00010101000000-000000000000
	melato.org/cloudconfiglxd v0.0.0-00010101000000-000000000000
	melato.org/command v0.0.0-20220222082143-ca56ccecf080
	melato.org/script v1.0.0
	melato.org/table3 v0.0.0-20210412105118-d2834489865c
	melato.org/yaml v0.0.0-00010101000000-000000000000
)

require (
	github.com/flosch/pongo2 v0.0.0-20200913210552-0d938eb266f3 // indirect
	github.com/go-macaroon-bakery/macaroon-bakery/v3 v3.0.1 // indirect
	github.com/go-macaroon-bakery/macaroonpb v1.0.0 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/juju/go4 v0.0.0-20160222163258-40d72ab9641a // indirect
	github.com/juju/persistent-cookiejar v1.0.0 // indirect
	github.com/juju/schema v1.0.1 // indirect
	github.com/juju/webbrowser v1.0.0 // indirect
	github.com/julienschmidt/httprouter v1.3.0 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/pborman/uuid v1.2.1 // indirect
	github.com/pkg/sftp v1.13.5 // indirect
	github.com/pkg/xattr v0.4.9 // indirect
	github.com/robfig/cron/v3 v3.0.1 // indirect
	github.com/rogpeppe/fastuuid v1.2.0 // indirect
	github.com/sirupsen/logrus v1.9.0 // indirect
	golang.org/x/crypto v0.5.0 // indirect
	golang.org/x/net v0.5.0 // indirect
	golang.org/x/sys v0.4.0 // indirect
	golang.org/x/term v0.4.0 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/errgo.v1 v1.0.1 // indirect
	gopkg.in/httprequest.v1 v1.2.1 // indirect
	gopkg.in/juju/environschema.v1 v1.0.1 // indirect
	gopkg.in/macaroon.v2 v2.1.0 // indirect
	gopkg.in/retry.v1 v1.0.3 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)
