# Blacklist Processor Plugin

The `blacklist` plugin reads in a restriction configuration file. The restriction file is an 
xml file which can contain both blacklists and whitelists.

If a table is in a blacklist, then it is dropped.
If a table is not in a whitelist, then it is dropped.

### Configuration:

```toml
[[processors.blacklist]]
  config = "/usr/local/akamai/goblin_telegraf/conf/goblin.restriction.conf"
```

## Restriction Configuration File:

```
<goblin_telegraf>
  <group owner="MapNocc">
    <criteria network="freeflow" />
    <criteria network="essl" />
    <whitelist>
      <table tablename="gm_example_table" />
    </whitelist>
  </group>
</goblin_telegraf>
```