# =======================================================
# 	common variable
# =======================================================
# console target log
log_target = echo -e "\033[0;32m===========> Running $@ ... \033[0m"
SHELL := /bin/bash
go_version := 1.22.0
go_dir = /usr/local
env_profile = /etc/profile
k8s_version = v1.29.1

# define kind cluster name
ifeq ($(origin cluster_name),undefined)
cluster_name = dev
endif

# define kind cluster k8s image
ifeq ($(origin k8s_image),undefined)
k8s_image = kindest/node:$(k8s_version)@sha256:0c06baa545c3bb3fbd4828eb49b8b805f6788e18ce67bff34706ffa91866558b
endif

# define kind_version
ifeq ($(origin kind_version),undefined)
kind_version = v0.22.0
endif

# define machine arch
ifeq ($(shell uname -m),x86_64)
arch = linux-amd64
else
	ifeq ($(shell uname -m),aarch64)
	arch = linux-arm664
	else
		$(error unable get system arch)
	endif
endif

# source url
kind_bin := https://kind.sigs.k8s.io/dl/$(kind_version)/kind-$(arch)
go_pkg := https://go.dev/dl/go$(go_version).$(arch).tar.gz


ifeq ($(wildcard $(go_dir)),)
$(shell mkdir $(go_dir))
endif


# =======================================================
#  tools install
# =======================================================
.PHONY: install-go
install-go: go_install
	@$(log_target)
	$(RM) -f go$(go_version).$(arch).tar.gz

.PHONY: install-kind
install-kind:
	@$(log_target)
	curl -sLo kind $(kind_bin)
	chmod +x ./kind
	sudo mv ./kind /usr/local/bin/kind

.PHONY: install-kubectl
install-kubectl:
	@$(log_target)
	curl -sLo kubectl https://dl.k8s.io/release/$(k8s_version)/bin/$(subst -,/,$(arch))/kubectl
	install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
	$(RM) -f kubectl

# =======================================================
# 	cluster operation
# =======================================================
# 创建kind集群
# TODO: 我们还需要添加多节点构建方式
.PHONY: create-cluster
create-cluster:
	@$(log_target)
	kind create cluster --name $(cluster_name) --image $(k8s_image)

.PHONY: go_install
go_install:
	@$(log_target)
	curl -sLo go$(go_version).$(arch).tar.gz $(go_pkg)
	tar -zxvf go$(go_version).$(arch).tar.gz -C $(go_dir)
	echo export GOROOT=/usr/local/go >> $(env_profile)
	echo export GOPATH=~/go >> $(env_profile)
	echo export GO111MODULE=on >> $(env_profile)
	echo export 'PATH=$$PATH:$$GOROOT/bin:$$GOPATH/bin' >> $(env_profile)
    $(shell source /etc/profile)


# =======================================================
# 	clean
# =======================================================
# 清除kind集群
.PHONY: clean
clean:
	@$(log_target)