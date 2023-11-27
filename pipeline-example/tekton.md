

# 安装 Tekton 组件
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

# 暴露服务

```
kubectl port-forward --address 0.0.0.0 service/el-kurator-pipeline-listener 30000:8080 -n kurator-pipeline
```









# clean up

k delete ns kurator-pipeline