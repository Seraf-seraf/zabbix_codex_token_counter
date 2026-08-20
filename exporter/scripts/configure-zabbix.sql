BEGIN;

DO $$
DECLARE
	codex_host_id bigint;
	applications_group_id bigint;
	codex_host_group_id bigint;
	metric record;
	metric_id bigint;
BEGIN
	SELECT hostid
	INTO codex_host_id
	FROM hosts
	WHERE host = 'codex-wrapper'
	  AND flags = 0;

	IF codex_host_id IS NULL THEN
		SELECT COALESCE(MAX(hostid), 0) + 1
		INTO codex_host_id
		FROM hosts;

		INSERT INTO hosts (hostid, host, name, name_upper)
		VALUES (codex_host_id, 'codex-wrapper', 'Codex exporter', 'CODEX EXPORTER');
	END IF;

	SELECT groupid
	INTO applications_group_id
	FROM hstgrp
	WHERE name = 'Applications'
	  AND type = 0;

	IF applications_group_id IS NULL THEN
		SELECT COALESCE(MAX(groupid), 0) + 1
		INTO applications_group_id
		FROM hstgrp;

		INSERT INTO hstgrp (groupid, name, type)
		VALUES (applications_group_id, 'Applications', 0);
	END IF;

	IF NOT EXISTS (
		SELECT 1
		FROM hosts_groups
		WHERE hostid = codex_host_id
		  AND groupid = applications_group_id
	) THEN
		SELECT COALESCE(MAX(hostgroupid), 0) + 1
		INTO codex_host_group_id
		FROM hosts_groups;

		INSERT INTO hosts_groups (hostgroupid, hostid, groupid)
		VALUES (codex_host_group_id, codex_host_id, applications_group_id);
	END IF;

	FOR metric IN
		SELECT *
		FROM (
			VALUES
				('Codex input tokens', 'codex.tokens.input'),
				('Codex cached input tokens', 'codex.tokens.cached_input'),
				('Codex cache write input tokens', 'codex.tokens.cache_write_input'),
				('Codex output tokens', 'codex.tokens.output'),
				('Codex reasoning output tokens', 'codex.tokens.reasoning_output'),
				('Codex total tokens', 'codex.tokens.total')
		) AS metrics(name, key_)
	LOOP
		IF NOT EXISTS (
			SELECT 1
			FROM items
			WHERE hostid = codex_host_id
			  AND key_ = metric.key_
		) THEN
			SELECT COALESCE(MAX(itemid), 0) + 1
			INTO metric_id
			FROM items;

			INSERT INTO items (
				itemid,
				type,
				hostid,
				name,
				key_,
				history,
				trends,
				value_type,
				units
			)
			VALUES (
				metric_id,
				2,
				codex_host_id,
				metric.name,
				metric.key_,
				'31d',
				'365d',
				3,
				'!tokens'
			);
		END IF;
	END LOOP;
END
$$;

COMMIT;
