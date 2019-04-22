## Dusk Wallet



## Cryptographic Primitives and Assumptions


- All scalars are assumed to be within the subgroup of the curve used. This assumption is justified as we are using ristretto, where we can think of the co-factor as being 1.

- Assume `G` to be the standard basepoint and `H` to be some other generator such that we do not know the discrete log, with respects to `G`.

- Pedersen Commitment is a point such that P = aG + bH

- HashToGroup; given arbitrary data, map it to some element in that group.

### Keys 

**Private** Spend Key:

    - seed = Rand()
    - PrivateSpendKey = HashToScalar(seed)

**Public** Spend Key:

    - PublicSpendKey = PrivateSpendKey * BasePoint


**Private** View Key:

    - PrivateViewKey = HashToScalar(PrivateSpendKey.Bytes())

**Public** View Key:

    - PublicViewKey = PrivateViewKey * BasePoint



