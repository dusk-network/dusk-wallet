// TODO


- Add input:

newInput will take the commitment, txid, secret key to unlock the tx and key to sign commitment to zero
create fake commitment, store the blinder in the tx for now
We should do all this in sign
We then run the mlsag on it to create the proof for this input

NOTE: Unexported items in input and output should not be serialised