#!/bin/bash
set -e

if [ "$SEED_SHARDS" = "0" ]
then
    echo "You cannot seed for zero shards. Exiting."
    exit 1
fi

WORKSPACE="${SEEDER_WORKSPACE:=/tmp/seeds}/$JOB_NAME"

mkdir -p $WORKSPACE

pushd $WORKSPACE

echo "Downloading and extracting shard seeder"
## Download binaries
aws s3 cp "s3://${BINARIES_S3_BUCKET}/binaries/$SEED_COMMIT.tar.gz" ./binaries.tar.gz
tar -xvf binaries.tar.gz tools/shard-seeder/shard-seeder
rm -rf binaries.tar.gz

shard_range=$(expr 255 / $SEED_SHARDS + 1)

echo "Checking if we are dealing with an old or new shard-seeder"

set +e
USAGE_DIRECTIVES=$(tools/shard-seeder/shard-seeder)
set -e
OLD=0
if [[ "$USAGE_DIRECTIVES" == *"number of shards"* ]]; then
    echo "Generating seeds old fashioned style!"

    tools/shard-seeder/shard-seeder "$SEED_SHARDS" "$SEED_OUTPUTS" "$SEED_VALUE" "$SEED_WITCOMM" "$SEED_MODE"
    OLD=1
else
    echo "Generating shards with a fancy config file"
    rm -rf config.cfg
    if [ ! -z "$SEED_CONFIG_BASE64" ]; then
        echo "Decoding passed config file from SEED_CONFIG_BASE64"
        echo "$SEED_CONFIG_BASE64" | base64 --decode > config.cfg
    elif [ ! -z "$SEED_CONFIG_S3URI" ]; then
        echo "Downloading configuration from S3: $SEED_CONFIG_S3URI"
        aws s3 cp "$SEED_CONFIG_S3URI" ./config.cfg
    else
        echo "Generating config file locally - will not work for sentinel attestation code"
        # Generate it - doesn't work for sentinel attestations
        touch config.cfg
        if [ "$SEED_MODE" = "1" ]; then
            printf "2pc=1\n" >> config.cfg
            printf "coordinator_count=1\ncoordinator0_count=1\ncoordinator0_0_endpoint=\"127.0.0.1:80\"\ncoordinator0_0_raft_endpoint=\"127.0.0.1:80\"\n" >> config.cfg
            printf "sentinel_count=1\nsentinel0_endpoint=\"127.0.0.1:80\"\n" >> config.cfg
            printf "shard_count=$SEED_SHARDS\n" >> config.cfg
            for (( i=0; i<$SEED_SHARDS; i++ ))
            do
                shard_start=$(($i * ${shard_range}))
                shard_end=$((${shard_start} + ${shard_range} - 1))
                printf "shard${i}_start=${shard_start}\nshard${i}_end=${shard_end}\nshard${i}_count=1\nshard${i}_0_endpoint=\"127.0.0.1:80\"\nshard${i}_0_raft_endpoint=\"127.0.0.1:80\"\nshard${i}_0_readonly_endpoint=\"127.0.0.1:80\"\n" >> config.cfg
            done
        else
            printf "sentinel_count=1\nsentinel0_endpoint=\"127.0.0.1:80\"\n" >> config.cfg
            printf "archiver_count=1\narchiver0_endpoint=\"127.0.0.1:80\"\narchiver0_db=\"db\"\n" >> config.cfg
            printf "atomizer_count=1\natomizer0_endpoint=\"127.0.0.1:80\"\natomizer0_raft_endpoint=\"127.0.0.1:80\"\n" >> config.cfg
            printf "watchtower_count=1\nwatchtower0_endpoint=\"127.0.0.1:80\"\nwatchtower0_internal_endpoint=\"127.0.0.1:80\"\nwatchtower0_client_endpoint=\"127.0.0.1:80\"\n" >> config.cfg
            printf "shard_count=$SEED_SHARDS\n" >> config.cfg
            for (( i=0; i<$SEED_SHARDS; i++ ))
            do
                shard_start=$(($i * ${shard_range}))
                shard_end=$((${shard_start} + ${shard_range} - 1))
                printf "shard${i}_endpoint=\"127.0.0.1:80\"\nshard${i}_start=${shard_start}\nshard${i}_end=${shard_end}\nshard${i}_db=\"db\"\n" >> config.cfg
            done
        fi
        printf "seed_privkey=\"$SEED_PRIVATEKEY\"\nseed_value=$SEED_VALUE\nseed_from=0\nseed_to=$SEED_OUTPUTS\n" >> config.cfg
    fi
    echo "Starting shard seeder"
    tools/shard-seeder/shard-seeder config.cfg
fi
rm -rf tools
rm -rf config.cfg

echo "Shard range: $shard_range"

maketar() {
    base=$(basename "$1")
    target="${base}_${SEED_COMMIT}.tar"
    if [ "$OLD" = "0" ]
    then
        suffix="${base##*_}"
        shard_start=$(($suffix * ${shard_range}))
        shard_end=$((${shard_start} + ${shard_range} - 1))
        target="${base%_*}_${shard_start}_${shard_end}_${SEED_COMMIT}.tar"
    fi
    echo "Making tar [$target] of [$f]"
    tar -cf "$target" "$f"
    echo "Exit code from tar: $?"
    rm -rf $1
}
echo "Directory listing follows"
ls -sla
echo "Diskfree result follows"
df -h
echo "Making TAR archives of seeds"
if [ "$SEED_MODE" = "0" ]
then
    for f in */
    do
        maketar $f
    done
else
    for f in *
    do
        maketar $f
    done
fi

for f in *.tar
do
    aws s3 cp $f "s3://${BINARIES_S3_BUCKET}/shard-preseeds/$f"
done

if [ ! -z "$SEED_CONFIG_S3URI" ]; then
    aws s3 rm "$SEED_CONFIG_S3URI"
fi
popd

rm -rf $WORKSPACE/*
