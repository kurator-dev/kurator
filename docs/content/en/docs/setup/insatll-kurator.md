---
title: "Install Kurator Cli"
linkTitle: "Install Kurator Cli"
weight: 10
description: >
  Instructions on installing Kurator cli.
---

## Install from source

Download kurator source and run make kurator

```bash
git clone https://github.com/kurator-dev/kurator.git
cd kurator
make kurator
```

And then move the executable binary to your PATH.

## Install from release package

1. Go to [Kurator release](https://github.com/kurator-dev/kurator/releases) page to download the release package for your OS and extract.

    ```console
    curl -L https://github.com/kurator-dev/kurator/releases/download/{{< kurator-version >}}/kurator-{{< kurator-version >}}.tar.gz
    tar -zxvf kurator-{{< kurator-version >}}.tar.gz
    ```

1. Move to release package directory.

    ```console
    cd kurator-{{< kurator-version >}}
    ```

    kurator binary is under `bin/` directory, and cluster operator helm chart is under `charts` directory.

1. Add kurator to your PATH

    ```console
    export PATH=$PWD/bin:$PATH
    ```
