module github.com/hashicorp/enos

go 1.26

// https://github.com/ryancragun/go-cty/commit/fe11e2e2dc2e25104c2915b2c8e5b463bbf412bd
replace github.com/zclconf/go-cty => github.com/ryancragun/go-cty v0.0.0-20251030155132-fe11e2e2dc2e

require (
	github.com/Masterminds/semver/v3 v3.4.0
	github.com/aws/aws-sdk-go-v2/config v1.32.14
	github.com/google/uuid v1.6.0
	github.com/hashicorp/cli v1.1.7
	github.com/hashicorp/go-hclog v1.6.3
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hashicorp/hcl/v2 v2.24.0
	github.com/hashicorp/terraform-exec v0.25.0
	github.com/hashicorp/terraform-json v0.27.2
	github.com/hexops/gotextdiff v1.0.3
	github.com/mitchellh/colorstring v0.0.0-20190213212951-d06e56a500db
	github.com/mitchellh/go-wordwrap v1.0.1
	github.com/olekukonko/tablewriter v0.0.5
	github.com/spf13/cobra v1.10.2
	github.com/stretchr/testify v1.11.1
	github.com/zclconf/go-cty v1.18.0
	github.com/zclconf/go-cty-yaml v1.2.0
	golang.org/x/sys v0.42.0
	golang.org/x/term v0.41.0
	golang.org/x/text v0.35.0
	google.golang.org/grpc v1.80.0
	google.golang.org/protobuf v1.36.11
)

require (
	dario.cat/mergo v1.0.2 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/sprig/v3 v3.3.0 // indirect
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/armon/go-radix v1.0.0 // indirect
	github.com/aws/aws-sdk-go-v2 v1.41.5 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.19.14 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.21 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.21 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.21 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.21 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.0.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.30.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.19 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.41.10 // indirect
	github.com/aws/smithy-go v1.24.3 // indirect
	github.com/bgentry/speakeasy v0.2.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fatih/color v1.19.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-version v1.9.0 // indirect
	github.com/huandu/xstrings v1.5.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.22 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/posener/complete v1.2.3 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	golang.org/x/crypto v0.49.0 // indirect
	golang.org/x/mod v0.34.0 // indirect
	golang.org/x/net v0.52.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/tools v0.43.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260406154035-8fb7ec149431 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
