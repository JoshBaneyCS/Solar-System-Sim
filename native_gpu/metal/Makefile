CC = clang
METAL = xcrun -sdk macosx metal
METALLIB = xcrun -sdk macosx metallib
COMMON = ../common

OBJC_FLAGS = -O2 -I$(COMMON) -fobjc-arc
C_FLAGS = -O2 -I$(COMMON)
LDFLAGS = -dynamiclib -framework Metal -framework Foundation -framework CoreGraphics

TARGET = libnative_render_metal.dylib

all: $(TARGET)

# Metal shader compilation requires full Xcode (not just Command Line Tools).
# If available, pre-compile for faster startup. Otherwise, shader is compiled
# from source at runtime using MTLDevice newLibraryWithSource.
metallib: raytracer.metallib

raytracer.air: raytracer.metal
	$(METAL) -c raytracer.metal -o raytracer.air

raytracer.metallib: raytracer.air
	$(METALLIB) raytracer.air -o raytracer.metallib

# Compile C sources (no ARC needed)
camera.o: $(COMMON)/camera.c
	$(CC) $(C_FLAGS) -c -o $@ $<

rasterizer.o: $(COMMON)/rasterizer.c
	$(CC) $(C_FLAGS) -c -o $@ $<

texture_loader.o: $(COMMON)/texture_loader.c
	$(CC) $(C_FLAGS) -c -o $@ $<

# Compile Objective-C sources (with ARC)
renderer.o: renderer.m
	$(CC) $(OBJC_FLAGS) -c -o $@ $<

hardware.o: hardware.m
	$(CC) $(OBJC_FLAGS) -c -o $@ $<

$(TARGET): renderer.o hardware.o camera.o rasterizer.o texture_loader.o
	$(CC) $(LDFLAGS) -o $@ $^

clean:
	rm -f $(TARGET) raytracer.air raytracer.metallib *.o

.PHONY: all clean
