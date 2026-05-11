package main

const maxShaderSpheres = 32

const rayShaderSource = `
package main

const MaxSpheres = 32
const MaxDepth = 4
const Epsilon = 0.001
const RefractionStrength = 0.35
const SecondaryRayLightStrength = 0.25

var Resolution vec2
var CamPos vec3
var CamYaw float
var CamPitch float

var SphereCount float
var SphereCenterRadius [32]vec4
var SphereAlbedoReflect [32]vec4
var SphereSpecShineTransIOR [32]vec4

var FloorPoint vec3
var FloorNormal vec3
var FloorColorA vec3
var FloorColorB vec3
var FloorParams vec4

var LightPosition vec3
var LightColor vec3
var LightIntensity float
var Ambient vec3

func clamp01(x float) float {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

func skyColor(dir vec3) vec3 {
	t := clamp01(dir.y*0.5 + 0.5)
	horizon := vec3(0.53, 0.80, 0.98)
	zenith := vec3(0.02, 0.20, 0.55)
	return horizon*(1-t) + zenith*t
}

func reflectVec(v vec3, n vec3) vec3 {
	return v - n*(2*dot(v, n))
}

func refractVec(v vec3, n vec3, eta float) vec3 {
	cosTheta := min(dot(-v, n), 1.0)
	rOutPerp := (v + n*cosTheta) * eta
	discriminant := 1.0 - dot(rOutPerp, rOutPerp)
	if discriminant < 0 {
		return reflectVec(v, n)
	}
	rOutParallel := n * -sqrt(abs(discriminant))
	return rOutPerp + rOutParallel
}

func canRefract(v vec3, n vec3, eta float) bool {
	cosTheta := min(dot(-v, n), 1.0)
	rOutPerp := (v + n*cosTheta) * eta
	discriminant := 1.0 - dot(rOutPerp, rOutPerp)
	return discriminant >= 0
}

func schlick(cosine float, refIdx float) float {
	r0 := (1 - refIdx) / (1 + refIdx)
	r0 = r0 * r0
	return r0 + (1-r0)*pow(1-cosine, 5)
}

func sphereIntersect(origin vec3, dir vec3, sphere vec4, tMin float, tMax float) float {
	center := sphere.xyz
	radius := sphere.w
	oc := origin - center
	a := dot(dir, dir)
	halfB := dot(oc, dir)
	c := dot(oc, oc) - radius*radius
	discriminant := halfB*halfB - a*c
	if discriminant < 0 {
		return -1
	}

	sqrtD := sqrt(discriminant)
	t := (-halfB - sqrtD) / a
	if t >= tMin && t <= tMax {
		return t
	}

	t = (-halfB + sqrtD) / a
	if t >= tMin && t <= tMax {
		return t
	}

	return -1
}

func planeIntersect(origin vec3, dir vec3, point vec3, normal vec3, tMin float, tMax float) float {
	n := normalize(normal)
	denom := dot(n, dir)
	if abs(denom) < 0.000001 {
		return -1
	}

	t := dot(point-origin, n) / denom
	if t < tMin || t > tMax {
		return -1
	}
	return t
}

func checkerColor(pos vec3) vec3 {
	scale := FloorParams.w
	if scale <= 0 {
		scale = 1
	}

	x := int(floor(pos.x * scale))
	z := int(floor(pos.z * scale))
	if (x+z)%2 == 0 {
		return FloorColorA
	}
	return FloorColorB
}

func isInShadow(point vec3, normal vec3) bool {
	toLight := LightPosition - point
	lightDistance := length(toLight)
	lightDir := normalize(toLight)
	shadowOrigin := point + normal*Epsilon

	for i := 0; i < MaxSpheres; i++ {
		if float(i) >= SphereCount {
			break
		}
		t := sphereIntersect(shadowOrigin, lightDir, SphereCenterRadius[i], Epsilon, lightDistance-Epsilon)
		if t > 0 {
			return true
		}
	}

	return false
}

func oneBounceColor(origin vec3, dir vec3) vec3 {
	closest := 1000000.0
	hitKind := 0
	hitSphere := 0

	for i := 0; i < MaxSpheres; i++ {
		if float(i) >= SphereCount {
			break
		}
		t := sphereIntersect(origin, dir, SphereCenterRadius[i], Epsilon, closest)
		if t > 0 {
			closest = t
			hitKind = 1
			hitSphere = i
		}
	}

	tPlane := planeIntersect(origin, dir, FloorPoint, FloorNormal, Epsilon, closest)
	if tPlane > 0 {
		closest = tPlane
		hitKind = 2
	}

	if hitKind == 0 {
		return skyColor(dir) * SecondaryRayLightStrength
	}

	hitPos := origin + dir*closest
	normal := vec3(0, 1, 0)
	baseColor := vec3(1, 0, 1)
	specular := 0.0
	shininess := 16.0

	if hitKind == 1 {
		sphere := SphereCenterRadius[hitSphere]
		materialA := SphereAlbedoReflect[hitSphere]
		materialB := SphereSpecShineTransIOR[hitSphere]

		normal = normalize(hitPos - sphere.xyz)
		baseColor = materialA.xyz
		specular = materialB.x
		shininess = materialB.y
	} else {
		normal = normalize(FloorNormal)
		if dot(normal, dir) > 0 {
			normal = -normal
		}
		baseColor = checkerColor(hitPos)
		specular = FloorParams.y
		shininess = FloorParams.z
	}

	ambient := baseColor * Ambient
	diffuse := vec3(0, 0, 0)
	spec := vec3(0, 0, 0)

	if !isInShadow(hitPos, normal) {
		lightDir := normalize(LightPosition - hitPos)
		ndotl := max(0, dot(normal, lightDir))
		diffuse = baseColor * LightColor * (ndotl * LightIntensity)

		if specular > 0 {
			viewDir := normalize(-dir)
			halfDir := normalize(lightDir + viewDir)
			specPower := pow(max(0, dot(normal, halfDir)), shininess)
			spec = LightColor * (specPower * specular * LightIntensity)
		}
	}

	return ambient + diffuse + spec
}

func sceneColor(origin vec3, dir vec3) vec3 {
	accumulated := vec3(0, 0, 0)
	attenuation := vec3(1, 1, 1)
	rayOrigin := origin
	rayDir := dir

	for depth := 0; depth < MaxDepth; depth++ {
		closest := 1000000.0
		hitKind := 0
		hitSphere := 0

		for i := 0; i < MaxSpheres; i++ {
			if float(i) >= SphereCount {
				break
			}
			t := sphereIntersect(rayOrigin, rayDir, SphereCenterRadius[i], Epsilon, closest)
			if t > 0 {
				closest = t
				hitKind = 1
				hitSphere = i
			}
		}

		tPlane := planeIntersect(rayOrigin, rayDir, FloorPoint, FloorNormal, Epsilon, closest)
		if tPlane > 0 {
			closest = tPlane
			hitKind = 2
		}

		if hitKind == 0 {
			sky := skyColor(rayDir)
			if depth > 0 {
				sky *= SecondaryRayLightStrength
			}
			accumulated += attenuation * sky
			break
		}

		hitPos := rayOrigin + rayDir*closest
		normal := vec3(0, 1, 0)
		baseColor := vec3(1, 0, 1)
		reflectivity := 0.0
		specular := 0.0
		shininess := 16.0
		transparency := 0.0
		refractionIndex := 1.0

		if hitKind == 1 {
			sphere := SphereCenterRadius[hitSphere]
			materialA := SphereAlbedoReflect[hitSphere]
			materialB := SphereSpecShineTransIOR[hitSphere]

			normal = normalize(hitPos - sphere.xyz)
			baseColor = materialA.xyz
			reflectivity = materialA.w
			specular = materialB.x
			shininess = materialB.y
			transparency = materialB.z
			refractionIndex = materialB.w
		} else {
			normal = normalize(FloorNormal)
			if dot(normal, rayDir) > 0 {
				normal = -normal
			}
			baseColor = checkerColor(hitPos)
			reflectivity = FloorParams.x
			specular = FloorParams.y
			shininess = FloorParams.z
		}

		ambient := baseColor * Ambient
		diffuse := vec3(0, 0, 0)
		spec := vec3(0, 0, 0)

		if !isInShadow(hitPos, normal) {
			lightDir := normalize(LightPosition - hitPos)
			ndotl := max(0, dot(normal, lightDir))
			diffuse = baseColor * LightColor * (ndotl * LightIntensity)

			if specular > 0 {
				viewDir := normalize(-rayDir)
				halfDir := normalize(lightDir + viewDir)
				specPower := pow(max(0, dot(normal, halfDir)), shininess)
				spec = LightColor * (specPower * specular * LightIntensity)
			}
		}

		localColor := ambient + diffuse + spec

		if transparency > 0 {
			outNormal := normal
			refRatio := 1.0 / max(refractionIndex, 0.0001)
			if dot(rayDir, normal) > 0 {
				outNormal = -normal
				refRatio = refractionIndex
			}

			refRatio = 1.0 + (refRatio-1.0)*RefractionStrength
			cosTheta := min(dot(-rayDir, outNormal), 1.0)
			reflectance := max(schlick(cosTheta, refRatio), reflectivity)
			reflectDir := normalize(reflectVec(rayDir, outNormal))
			refractDir := normalize(refractVec(rayDir, outNormal, refRatio))
			reflectedColor := oneBounceColor(hitPos+outNormal*Epsilon, reflectDir)

			if !canRefract(rayDir, outNormal, refRatio) {
				accumulated += attenuation * localColor * (1 - transparency)
				attenuation *= transparency
				rayOrigin = hitPos + outNormal*Epsilon
				rayDir = reflectDir
			} else {
				accumulated += attenuation * (localColor*(1-transparency) + reflectedColor*(transparency*reflectance))
				attenuation *= transparency * (1 - reflectance)
				rayOrigin = hitPos - outNormal*Epsilon
				rayDir = refractDir
			}
		} else if reflectivity > 0.01 {
			accumulated += attenuation * localColor * (1 - reflectivity)
			attenuation *= reflectivity
			rayDir = normalize(reflectVec(rayDir, normal))
			rayOrigin = hitPos + normal*Epsilon
		} else {
			accumulated += attenuation * localColor
			break
		}
	}

	return accumulated
}

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	uv := position.xy / Resolution
	fov := 3.1415926535 / 3.0
	aspectRatio := Resolution.x / Resolution.y
	scale := tan(fov / 2.0)

	px := (2.0*(uv.x) - 1.0) * aspectRatio * scale
	py := (1.0 - 2.0*(uv.y)) * scale

	cosYaw := cos(CamYaw)
	sinYaw := sin(CamYaw)
	cosPitch := cos(CamPitch)
	sinPitch := sin(CamPitch)

	localX := px
	localY := py
	localZ := 1.0

	y1 := localY*cosPitch - localZ*sinPitch
	z1 := localY*sinPitch + localZ*cosPitch

	dirX := localX*cosYaw + z1*sinYaw
	dirY := y1
	dirZ := -localX*sinYaw + z1*cosYaw
	rayDir := normalize(vec3(dirX, dirY, dirZ))

	col := sceneColor(CamPos, rayDir)
	col = vec3(sqrt(clamp01(col.x)), sqrt(clamp01(col.y)), sqrt(clamp01(col.z)))
	return vec4(col, 1)
}
`
