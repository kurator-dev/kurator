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

Go to [Kurator release](https://github.com/kurator-dev/kurator/releases) page to download the release package for your OS and extract.

```console
curl -LO https://github.com/kurator-dev/kurator/releases/download/v{{< kurator-version >}}/kurator-{{< kurator-version >}}-linux-amd64.tar.gz
tar -zxvf kurator-{{< kurator-version >}}-linux-amd64.tar.gz
```

kurator binary is in the current directory, move it to your user PATH
