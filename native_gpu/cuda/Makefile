NVCC = nvcc
COMMON = ../common

# sm_70 covers Volta+ (V100, RTX 20xx, 30xx, 40xx, etc.)
NVCC_FLAGS = -O3 -arch=sm_70 --compiler-options '-fPIC' -I$(COMMON)

TARGET = libnative_render_cuda.so

# C sources compiled by host compiler through nvcc
C_SOURCES = $(COMMON)/camera.c \
            $(COMMON)/rasterizer.c \
            $(COMMON)/texture_loader.c

all: $(TARGET)

$(TARGET): raytracer.cu renderer.cu hardware.cu $(C_SOURCES)
	$(NVCC) -shared $(NVCC_FLAGS) -o $@ $^ -lcudart

clean:
	rm -f $(TARGET)

.PHONY: all clean
