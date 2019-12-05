truncate table hosts;

insert into hosts(guid, ipaddress, telemetryid)
select distinct guid, address, name from connections 
where address <> ''
group by guid, address, name;

update hosts
set srvname = relaytelemetryid.relay
from relaytelemetryid
where relaytelemetryid.telemetryid = hosts.telemetryid;

