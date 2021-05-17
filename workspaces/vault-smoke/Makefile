default: init run destroy

init: verify-license-init upgrade-init
run: verify-license-run upgrade-run
destroy: verify-license-destroy upgrade-destroy

verify-license-init:
	export TF_CLI_CONFIG_FILE=./mirror.tfrc &&\
	mkdir -p ./terraform-plugin-cache && \
	terraform -chdir=./scenarios/verify-license init -upgrade -var-file=../../terraform.tfvars

verify-license-run:
	export TF_CLI_CONFIG_FILE=./mirror.tfrc &&\
	terraform -chdir=./scenarios/verify-license apply -var-file=../../terraform.tfvars -auto-approve

verify-license-destroy:
	export TF_CLI_CONFIG_FILE=./mirror.tfrc &&\
	terraform -chdir=./scenarios/verify-license destroy -var-file=../../terraform.tfvars -auto-approve

upgrade-init:
	export TF_CLI_CONFIG_FILE=./mirror.tfrc &&\
	mkdir -p ./terraform-plugin-cache && \
	terraform -chdir=./scenarios/upgrade init -upgrade -var-file=../../terraform.tfvars

upgrade-run:
	export TF_CLI_CONFIG_FILE=./mirror.tfrc &&\
	terraform -chdir=./scenarios/upgrade apply -var-file=../../terraform.tfvars -auto-approve

upgrade-destroy:
	export TF_CLI_CONFIG_FILE=./mirror.tfrc &&\
	terraform -chdir=./scenarios/upgrade destroy -var-file=../../terraform.tfvars -auto-approve
