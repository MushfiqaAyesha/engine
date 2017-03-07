// Copyright 2016 The G3N Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package graphic

import (
	"github.com/g3n/engine/core"
	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"
)

// Lines is a Graphic which is rendered as a collection of independent lines
type Lines struct {
	Graphic
	mvpm gls.UniformMatrix4f // Model view projection matrix uniform
}

func (l *Lines) Init(igeom geometry.IGeometry, imat material.IMaterial) {

	l.Graphic.Init(igeom, gls.LINES)
	l.AddMaterial(l, imat, 0, 0)
	l.mvpm.Init("MVP")
}

func NewLines(igeom geometry.IGeometry, imat material.IMaterial) *Lines {

	l := new(Lines)
	l.Init(igeom, imat)
	return l
}

// RenderSetup is called by the engine before drawing this geometry
func (l *Lines) RenderSetup(gs *gls.GLS, rinfo *core.RenderInfo) {

	// Calculates model view projection matrix and updates uniform
	mw := l.MatrixWorld()
	var mvpm math32.Matrix4
	mvpm.MultiplyMatrices(&rinfo.ViewMatrix, &mw)
	mvpm.MultiplyMatrices(&rinfo.ProjMatrix, &mvpm)
	l.mvpm.SetMatrix4(&mvpm)
	l.mvpm.Transfer(gs)
}

// Raycast satisfies the INode interface and checks the intersections
// of this geometry with the specified raycaster
func (l *Lines) Raycast(rc *core.Raycaster, intersects *[]core.Intersect) {

	lineRaycast(l, rc, intersects, 2)
}

// Internal function used by raycasting for Lines and LineStrip
func lineRaycast(igr IGraphic, rc *core.Raycaster, intersects *[]core.Intersect, step int) {

	// Get the bounding sphere
	gr := igr.GetGraphic()
	geom := igr.GetGeometry()
	sphere := geom.BoundingSphere()

	// Transform bounding sphere from model to world coordinates and
	// checks intersection with raycaster
	matrixWorld := gr.MatrixWorld()
	sphere.ApplyMatrix4(&matrixWorld)
	if !rc.IsIntersectionSphere(&sphere) {
		return
	}

	// Copy ray and transform to model coordinates
	// This ray will will also be used to check intersects with
	// the geometry, as is much less expensive to transform the
	// ray to model coordinates than the geometry to world coordinates.
	var inverseMatrix math32.Matrix4
	var ray math32.Ray
	inverseMatrix.GetInverse(&matrixWorld, true)
	ray.Copy(&rc.Ray).ApplyMatrix4(&inverseMatrix)

	var vstart math32.Vector3
	var vend math32.Vector3
	var interSegment math32.Vector3
	var interRay math32.Vector3

	// Get geometry positions and indices buffers
	vboPos := geom.VBO("VertexPosition")
	if vboPos == nil {
		return
	}
	positions := vboPos.Buffer()
	indices := geom.Indices()
	precisionSq := rc.LinePrecision * rc.LinePrecision

	// Checks intersection with individual lines for indexed geometry
	if indices.Size() > 0 {
		for i := 0; i < indices.Size()-1; i += step {
			// Calculates distance from ray to this line segment
			a := indices[i]
			b := indices[i+1]
			positions.GetVector3(int(3*a), &vstart)
			positions.GetVector3(int(3*b), &vend)
			distSq := ray.DistanceSqToSegment(&vstart, &vend, &interRay, &interSegment)
			if distSq > precisionSq {
				continue
			}
			// Move back to world coordinates for distance calculation
			interRay.ApplyMatrix4(&matrixWorld)
			origin := rc.Ray.Origin()
			distance := origin.DistanceTo(&interRay)
			if distance < rc.Near || distance > rc.Far {
				continue
			}

			interSegment.ApplyMatrix4(&matrixWorld)
			*intersects = append(*intersects, core.Intersect{
				Distance: distance,
				Point:    interSegment,
				Index:    uint32(i),
				Object:   igr,
			})
		}
		// Checks intersection with individual lines for NON indexed geometry
	} else {
		for i := 0; i < positions.Size()/3-1; i += step {
			positions.GetVector3(int(3*i), &vstart)
			positions.GetVector3(int(3*i+3), &vend)
			distSq := ray.DistanceSqToSegment(&vstart, &vend, &interRay, &interSegment)
			if distSq > precisionSq {
				continue
			}

			// Move back to world coordinates for distance calculation
			interRay.ApplyMatrix4(&matrixWorld)
			origin := rc.Ray.Origin()
			distance := origin.DistanceTo(&interRay)
			if distance < rc.Near || distance > rc.Far {
				continue
			}

			interSegment.ApplyMatrix4(&matrixWorld)
			*intersects = append(*intersects, core.Intersect{
				Distance: distance,
				Point:    interSegment,
				Index:    uint32(i),
				Object:   igr,
			})
		}
	}
}
