extraArgs=""

for arg in $@; do
  extraArgs+=" $arg "
done

./scripts/create-testnet.sh mode=normal --light-computations $extraArgs