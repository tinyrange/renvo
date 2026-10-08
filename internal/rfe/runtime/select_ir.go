package runtime

// SelectValue uses A as condition, B as the true value and Imm as the backward
// SSA index of the false value. Both alternatives are already-computed values;
// this operation neither speculates nor reorders a memory effect.
const SelectValue = 21
