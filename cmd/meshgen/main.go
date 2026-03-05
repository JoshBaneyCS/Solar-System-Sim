package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
)

func main() {
	segments := flag.Int("segments", 32, "Number of latitude/longitude segments")
	output := flag.String("output", "sphere.glb", "Output .glb file path")
	flag.Parse()

	if *segments < 4 {
		fmt.Fprintln(os.Stderr, "segments must be >= 4")
		os.Exit(1)
	}

	positions, normals, uvs, indices := generateUVSphere(*segments)

	glb, err := buildGLB(positions, normals, uvs, indices)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error building GLB: %v\n", err)
		os.Exit(1)
	}

	dir := filepath.Dir(*output)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
			os.Exit(1)
		}
	}

	if err := os.WriteFile(*output, glb, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}

	vertCount := (*segments + 1) * (*segments + 1)
	triCount := len(indices) / 3
	fmt.Printf("Generated %s: %d vertices, %d triangles, %d segments\n", *output, vertCount, triCount, *segments)
}

func generateUVSphere(segments int) (positions, normals, uvs []float32, indices []uint16) {
	for lat := 0; lat <= segments; lat++ {
		theta := float64(lat) * math.Pi / float64(segments)
		sinTheta := math.Sin(theta)
		cosTheta := math.Cos(theta)

		for lon := 0; lon <= segments; lon++ {
			phi := float64(lon) * 2.0 * math.Pi / float64(segments)
			sinPhi := math.Sin(phi)
			cosPhi := math.Cos(phi)

			x := float32(sinTheta * cosPhi)
			y := float32(cosTheta)
			z := float32(sinTheta * sinPhi)
			u := float32(lon) / float32(segments)
			v := float32(lat) / float32(segments)

			positions = append(positions, x, y, z)
			normals = append(normals, x, y, z) // unit sphere: normal = position
			uvs = append(uvs, u, v)
		}
	}

	stride := segments + 1
	for lat := 0; lat < segments; lat++ {
		for lon := 0; lon < segments; lon++ {
			a := uint16(lat*stride + lon)
			b := uint16(a + 1)
			c := uint16((lat+1)*stride + lon)
			d := uint16(c + 1)

			if lat != 0 {
				indices = append(indices, a, c, b)
			}
			if lat != segments-1 {
				indices = append(indices, b, c, d)
			}
		}
	}

	return
}

func buildGLB(positions, normals, uvs []float32, indices []uint16) ([]byte, error) {
	// Build binary buffer
	var binBuf bytes.Buffer

	// Write indices
	idxOffset := 0
	for _, idx := range indices {
		binary.Write(&binBuf, binary.LittleEndian, idx)
	}
	idxSize := binBuf.Len()

	// Pad to 4-byte alignment
	for binBuf.Len()%4 != 0 {
		binBuf.WriteByte(0)
	}

	// Write positions
	posOffset := binBuf.Len()
	for _, v := range positions {
		binary.Write(&binBuf, binary.LittleEndian, v)
	}
	posSize := binBuf.Len() - posOffset

	// Write normals
	normOffset := binBuf.Len()
	for _, v := range normals {
		binary.Write(&binBuf, binary.LittleEndian, v)
	}
	normSize := binBuf.Len() - normOffset

	// Write UVs
	uvOffset := binBuf.Len()
	for _, v := range uvs {
		binary.Write(&binBuf, binary.LittleEndian, v)
	}
	uvSize := binBuf.Len() - uvOffset

	vertCount := len(positions) / 3
	idxCount := len(indices)

	// Compute position bounds
	var minPos, maxPos [3]float32
	for i := 0; i < 3; i++ {
		minPos[i] = math.MaxFloat32
		maxPos[i] = -math.MaxFloat32
	}
	for i := 0; i < len(positions); i += 3 {
		for j := 0; j < 3; j++ {
			if positions[i+j] < minPos[j] {
				minPos[j] = positions[i+j]
			}
			if positions[i+j] > maxPos[j] {
				maxPos[j] = positions[i+j]
			}
		}
	}

	gltf := map[string]interface{}{
		"asset": map[string]string{
			"version":   "2.0",
			"generator": "solar-system-sim meshgen",
		},
		"scene": 0,
		"scenes": []map[string]interface{}{
			{"nodes": []int{0}},
		},
		"nodes": []map[string]interface{}{
			{"mesh": 0},
		},
		"meshes": []map[string]interface{}{
			{
				"primitives": []map[string]interface{}{
					{
						"attributes": map[string]int{
							"POSITION":   1,
							"NORMAL":     2,
							"TEXCOORD_0": 3,
						},
						"indices": 0,
					},
				},
			},
		},
		"accessors": []map[string]interface{}{
			// 0: indices
			{
				"bufferView":    0,
				"componentType": 5123, // UNSIGNED_SHORT
				"count":         idxCount,
				"type":          "SCALAR",
			},
			// 1: positions
			{
				"bufferView":    1,
				"componentType": 5126, // FLOAT
				"count":         vertCount,
				"type":          "VEC3",
				"min":           []float32{minPos[0], minPos[1], minPos[2]},
				"max":           []float32{maxPos[0], maxPos[1], maxPos[2]},
			},
			// 2: normals
			{
				"bufferView":    2,
				"componentType": 5126,
				"count":         vertCount,
				"type":          "VEC3",
			},
			// 3: UVs
			{
				"bufferView":    3,
				"componentType": 5126,
				"count":         vertCount,
				"type":          "VEC2",
			},
		},
		"bufferViews": []map[string]interface{}{
			// 0: indices
			{
				"buffer":     0,
				"byteOffset": idxOffset,
				"byteLength": idxSize,
				"target":     34963, // ELEMENT_ARRAY_BUFFER
			},
			// 1: positions
			{
				"buffer":     0,
				"byteOffset": posOffset,
				"byteLength": posSize,
				"target":     34962, // ARRAY_BUFFER
			},
			// 2: normals
			{
				"buffer":     0,
				"byteOffset": normOffset,
				"byteLength": normSize,
				"target":     34962,
			},
			// 3: UVs
			{
				"buffer":     0,
				"byteOffset": uvOffset,
				"byteLength": uvSize,
				"target":     34962,
			},
		},
		"buffers": []map[string]interface{}{
			{
				"byteLength": binBuf.Len(),
			},
		},
	}

	jsonData, err := json.Marshal(gltf)
	if err != nil {
		return nil, err
	}

	// Pad JSON to 4-byte alignment
	for len(jsonData)%4 != 0 {
		jsonData = append(jsonData, ' ')
	}

	binData := binBuf.Bytes()
	// Pad binary to 4-byte alignment
	for len(binData)%4 != 0 {
		binData = append(binData, 0)
	}

	totalLen := 12 + 8 + len(jsonData) + 8 + len(binData)

	var out bytes.Buffer
	// GLB header
	out.Write([]byte("glTF"))
	binary.Write(&out, binary.LittleEndian, uint32(2))         // version
	binary.Write(&out, binary.LittleEndian, uint32(totalLen))   // total length

	// JSON chunk
	binary.Write(&out, binary.LittleEndian, uint32(len(jsonData)))
	binary.Write(&out, binary.LittleEndian, uint32(0x4E4F534A)) // "JSON"
	out.Write(jsonData)

	// Binary chunk
	binary.Write(&out, binary.LittleEndian, uint32(len(binData)))
	binary.Write(&out, binary.LittleEndian, uint32(0x004E4942)) // "BIN\0"
	out.Write(binData)

	return out.Bytes(), nil
}
