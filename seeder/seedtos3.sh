#!/bin/bash
set -e

if [ "$SEED_SHARDS" = "0" ]
then
    echo "You cannot seed for zero shards. Exiting."
    exit 1
fi

mkdir -p /tmp/seeds

cd /tmp/seeds

echo "Downloading and extracting shard seeder"
## Download binaries
aws s3 cp "s3://${BINARIES_S3_BUCKET}/binaries/$SEED_COMMIT.tar.gz" ./binaries.tar.gz
tar -xvf binaries.tar.gz tools/shard-seeder/shard-seeder
rm -rf binaries.tar.gz

echo "Generating seeds"
tools/shard-seeder/shard-seeder "$SEED_SHARDS" "$SEED_OUTPUTS" "$SEED_VALUE" "$SEED_WITCOMM" "$SEED_MODE"

rm -rf tools

echo "Making TAR archives of seeds"
if [ "$SEED_MODE" = "0" ]
then
    for f in */
    do
        base=$(basename "$f")
        tar -cf "${base}_${SEED_COMMIT}.tar" "$f"
        rm -rf $f
    done
else
    for f in *
    do
        base=$(basename "$f")
        tar -cf "${base}_${SEED_COMMIT}.tar" "$f"
        rm -rf $f
    done
fi

for f in *.tar
do

    aws s3 cp $f "s3://${BINARIES_S3_BUCKET}/shard-preseeds/$f"
done