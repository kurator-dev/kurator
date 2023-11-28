
私有仓库 secret

根据官方文档获取 个人的 token


kubectl create secret generic git-credentials -n kurator-pipeline \
--from-literal=.gitconfig='[credential "https://github.com"]\n\thelper = store' \
--from-literal=.git-credentials="https://Xieql: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@github.com"



cosign secret

