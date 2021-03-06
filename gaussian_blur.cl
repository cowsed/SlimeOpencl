__constant sampler_t sampler = CLK_NORMALIZED_COORDS_FALSE | CLK_ADDRESS_CLAMP_TO_EDGE | CLK_FILTER_NEAREST;




__kernel void gaussian_blur(
  __read_only image2d_t image,
  __write_only image2d_t blurredImage,
  const float fadeStr,
  const float diffuseStr,
  const unsigned int Width,
    const unsigned int Height
) {

  int id = get_global_id(0);
  int idx = id % Width;
  int idy = id / Width;

  const int2 pos = {
    idx,
    idy
  };

  float4 originalValue = read_imagef(image, sampler, pos);

  // Collect neighbor values and multiply with gaussian
  float4 sum = 0.0 ;

  for (int offsetX = -1; offsetX <= 1; offsetX++) {
    for (int offsetY = -1; offsetY <= 1; offsetY++) {
      int sampleX = idx+offsetX;
      int sampleY = idy+offsetY;
      sum += read_imagef(image, sampler,  (int2)(sampleX, sampleY));
    }
  }

  float4 blurResult=sum/(float4)(9.0);
  
  //Fade away

  float4 diffusedValue = mix(originalValue, blurResult, diffuseStr);
  float4 combinedResult=diffusedValue ;

  combinedResult.x=max(0.0f,combinedResult.x-fadeStr);
  combinedResult.y=max(0.0f,combinedResult.y-fadeStr);
  combinedResult.z=max(0.0f,combinedResult.z-fadeStr);

  blurResult.w=1;

  
  write_imagef(blurredImage, pos.xy, combinedResult);
}