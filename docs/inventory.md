---
title: "Inventory"
sidebar:
  order: 4
---

`datumctl inventory` is a read view over the Datum Cloud physical inventory —
the providers, regions, sites, clusters, and nodes that make up the real
infrastructure Datum Cloud runs on. Use it to answer questions like "which sites
are in this region?", "which provider owns this site?", and "which nodes are
assigned to this cluster?" without writing label selectors by hand.

Inventory records live on the platform root, so every `inventory` subcommand
defaults to `--platform-wide`. Pass `--organization` or `--project` to override
the scope.

## Listing resources

Each kind has its own list subcommand:

```
datumctl inventory providers
datumctl inventory regions
datumctl inventory sites
datumctl inventory clusters
datumctl inventory nodes
```

By default each prints a table with the most useful columns. Use `-o json` or
`-o yaml` for the full objects (handy for scripting):

```
datumctl inventory sites -o json
```

## Filtering

List subcommands accept filter flags that narrow the results by topology:

```
# Sites in one region
datumctl inventory sites --region us-central-2

# Sites from one provider
datumctl inventory sites --provider netactuate

# Nodes at a site, or assigned to a cluster
datumctl inventory nodes --site us-central-2a
datumctl inventory nodes --cluster my-edge-cluster

# Clusters in a region
datumctl inventory clusters --region us-central-2
```

Region, site, and cluster filters are resolved server-side using the
`topology.inventory.miloapis.com/*` labels that the platform propagates onto
inventory objects. The provider filter matches on the site's `providerRef`.

## Topology tree

`datumctl inventory tree` prints the region → site → node hierarchy, with the
clusters anchored in each region listed alongside:

```
datumctl inventory tree
datumctl inventory tree --region us-central-2
```

## Summary

`datumctl inventory summary` prints fleet-wide counts: totals per kind, sites
and nodes per region, and sites per provider.

```
datumctl inventory summary
```
