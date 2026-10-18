// Fixed benchmark actions only. The ARM candidate namespace is diagnostic/optimization scope.
import {readFile} from 'node:fs/promises';
import {main,validateArmTrial} from './coremark.mjs';
const [action,value,...extra]=process.argv.slice(2);
if(extra.length)throw Error('Unexpected benchmark arguments');
const source=await readFile(new URL('./coremark.mjs',import.meta.url),'utf8');
if(action?.startsWith('arm-')){
 const selected=action.slice(4);
 if(!['prepare','before-build','check','run','profile'].includes(selected))throw Error('Invalid ARM benchmark action');
 await main(selected,'aarch64',source,validateArmTrial(value));
}else{
 if(!['prepare','before-build','check','run'].includes(action)||!['aarch64','x86_64'].includes(value))throw Error('Invalid fixed benchmark action');
 await main(action,value,source);
}
