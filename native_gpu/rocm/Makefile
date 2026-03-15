HIPCC = hipcc
COMMON = ../common

HIP_FLAGS = -O3 -fPIC -I$(COMMON)

TARGET = libnative_render_rocm.so

C_SOURCES = $(COMMON)/camera.c \
            $(COMMON)/rasterizer.c \
            $(COMMON)/texture_loader.c

all: $(TARGET)

$(TARGET): raytracer.hip renderer.hip hardware.hip $(C_SOURCES)
	$(HIPCC) -shared $(HIP_FLAGS) -o $@ $^

clean:
	rm -f $(TARGET)

.PHONY: all clean
