const sampler_t samplerA = CLK_ADDRESS_REPEAT;


unsigned int hash(unsigned int state){
	state ^= 2737636419u;
	state *= 2654435769u;
	state ^= state >> 16;
	state *= 2654435769u;
	state ^= state >> 16;
	state *= 2654435769u;
	return state;
	//4294967295.0
}



typedef struct
{
  float2 pos;
  float dir;

} Agent;

__kernel void
simulate (__global float *agentData,
	  const unsigned int Width,
	  const unsigned int Height,
	  __read_only image2d_t image_in,
	  __write_only image2d_t image_out, const unsigned int count){
		  
  int id = get_global_id (0);
  if (id >= count)
    return;

  int index = id * 3;
  Agent agent ={ {agentData[index], agentData[index + 1]}, agentData[index + 2] };


  //Draw current Point
  float4 color = (float4) (1, 1, 1, 1);
  
  write_imagef (image_out, (int2) ((int) agent.pos.x, (int) agent.pos.y), color);


  //Change direction if rngesus decides so
  float dirChance = 0.5;
  float dirWindow = 0.4;
  unsigned int key = (unsigned int)id + (unsigned int)agent.pos.x + (unsigned int)agent.pos.y;

  float rand = (float) hash(key)/ 4294967295.0;
  if (rand <= dirChance ){
	rand/=dirChance * dirWindow;// make rand back between 0 and 1 then scale to be in the dir window
	agentData[index+2]+=rand*2*M_PI;
  } 
  
  //Move agents along
  agentData[index] += cos (agent.dir) * 1.5;
  agentData[index + 1] += sin (agent.dir) * 1.5;





}
