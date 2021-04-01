const sampler_t samplerA = CLK_ADDRESS_CLAMP_TO_EDGE ;


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

float4 sense (bool print, Agent a, float angle, image2d_t image_in, int Width, int Height){
  int sensorSize = 2;
  float senseDist = 4;
  float sensorAngle= a.dir+angle;
  float2 pos = a.pos+ senseDist * (float2)(cos(sensorAngle),sin(sensorAngle));
  if (print){
   // printf("New Pos(%f, %f) pointing %f ",pos.x,pos.y, sensorAngle);
  }
  float4 sum = {0,0,0,0};
  for (int offsetX = -sensorSize; offsetX <= sensorSize; offsetX++) {
    for (int offsetY = -sensorSize; offsetY <= sensorSize; offsetY++) {
      int sampleX = (int)pos.x+offsetX;
      int sampleY = (int)pos.y+offsetY;
       //if (sampleX>0 && sampleX<Width && sampleY>0 && sampleY<Height){
	sum+=read_imagef(image_in, samplerA,  (int2)(sampleX, sampleY));

	    //}
    }
  }

  //if (print) printf("found: %f\n",length(sum));
  return sum;
}

__kernel void
simulate (__global float *agentData,
	  const unsigned int Width,
	  const unsigned int Height,
	  const float dirChance, 
	  const float dirWindow,
	  const float SensorAngle,
	  const float Speed, 
	  __read_only image2d_t image_in,
	  __write_only image2d_t image_out, 
	  const unsigned int count,
	  const unsigned int frame){
		  
  int id = get_global_id (0);
  if (id >= count)
    return;

  if (id==0){
      //printf("wind %f, chance %f", dirWindow, dirChance);	  
  }
  
  int index = id * 3;
  Agent agent ={ {agentData[index], agentData[index + 1]}, agentData[index + 2] };


  //Draw current Point
  float4 color = (float4) (1, 1, 1, 1);
  
  write_imagef (image_out, (int2) ((int) agent.pos.x, (int) agent.pos.y), color);


  
  float sensorAngleSpacing = SensorAngle;//M_PI/4;
  bool draw = id==0;
  float4 senseFront = sense(draw, agent,0,image_in, Width, Height);
  float4 senseRight = sense(draw, agent,sensorAngleSpacing ,image_in, Width, Height);
  float4 senseLeft = sense(draw, agent,-sensorAngleSpacing ,image_in, Width, Height);

  
  //Change direction according to sense decides so
  unsigned int key = (unsigned int)id * frame;
  float rand = (float) hash(key)/ 4294967295.0;
  
  //Constiue Forward
  if (length(senseFront) > length(senseLeft )&& length(senseFront) > length(senseRight)){
    //agent.dir+=0;
  } 
  //Random
  else if (length(senseFront) < length(senseLeft )&& length(senseFront) < length(senseRight)){
    agent.dir+=rand*M_PI*2*dirWindow;
  } 
  //Right
  else if (length(senseRight)>length(senseLeft)){
    agent.dir-=dirWindow*2*M_PI;
  }
  //Left
  else if(length(senseLeft)>length(senseRight)){
    agent.dir+=dirWindow*2*M_PI;
  }

  
  
  //Move agents along
  agent.pos.x += cos (agent.dir) * Speed;
  agent.pos.y += sin (agent.dir) * Speed;


  
  
  int x = (int)agent.pos.x;
  int y = (int)agent.pos.y;
  float rA = rand*2*M_PI;
  if (x>=Width){
	agent.pos.x=(float)(Width-2);
	agent.dir=rA;
  } else if (x<=0){
	agent.pos.x=(float)(2);
	agent.dir=rA;
  }
  if (y>=Height){
        agent.pos.y = (float)(Height-2);
	agent.dir=rA;
  } 
  if (y<=0){
	agent.pos.y = 2;
	agent.dir=rA;
  }
  
  agentData[index] = agent.pos.x;
  agentData[index + 1] = agent.pos.y; 
  agentData[index + 2] = agent.dir;




}
