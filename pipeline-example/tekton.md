如何在当前集群安装所需的 tekton controller

安装 pipeline
kubectl apply --filename \
https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml


安装 trigger
kubectl apply --filename \
https://storage.googleapis.com/tekton-releases/triggers/latest/release.yaml
kubectl apply --filename \
https://storage.googleapis.com/tekton-releases/triggers/latest/interceptors.yaml

安装 chain
kubectl apply --filename \
https://storage.googleapis.com/tekton-releases/chains/latest/release.yaml


clean up

k delete ns kurator-pipeline