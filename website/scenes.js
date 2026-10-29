import * as THREE from './assets/three.module.js'

// 每个功能卡片的 3D 场景：发光方块、层叠结构、方块阵列、粒子流、线框环。

const PURPLE = 0xbd34fe
const BLUE = 0x41d1ff
const LIGHT = 0xffffff

function makeBox(size, color, opacity) {
  const group = new THREE.Group()

  const geo = new THREE.BoxGeometry(size, size, size)
  const solid = new THREE.Mesh(
    geo,
    new THREE.MeshBasicMaterial({ color, transparent: true, opacity }),
  )
  group.add(solid)

  const edges = new THREE.LineSegments(
    new THREE.EdgesGeometry(geo),
    new THREE.LineBasicMaterial({ color: LIGHT, transparent: true, opacity: 1 }),
  )
  group.add(edges)

  return group
}

function buildDevices(group) {
  // 两台“设备”：一大一小两个发光方块，中间一条连线
  const a = makeBox(1.5, PURPLE, 0.16)
  a.position.set(-1.5, 0.1, 0)
  group.add(a)

  const b = makeBox(1.0, BLUE, 0.16)
  b.position.set(1.6, -0.2, 0.3)
  group.add(b)

  const points = [
    new THREE.Vector3(-0.75, 0.1, 0),
    new THREE.Vector3(0, 0.6, 0),
    new THREE.Vector3(1.1, -0.2, 0.3),
  ]
  const curve = new THREE.CatmullRomCurve3(points)
  const line = new THREE.Line(
    new THREE.BufferGeometry().setFromPoints(curve.getPoints(40)),
    new THREE.LineBasicMaterial({ color: LIGHT, transparent: true, opacity: 0.55 }),
  )
  group.add(line)
}

function buildLayers(group) {
  // 三层叠放的扁方块，像部署层
  for (let i = 0; i < 3; i += 1) {
    const layer = makeBox(2.2, i === 1 ? BLUE : PURPLE, 0.12)
    layer.scale.y = 0.14
    layer.position.y = (i - 1) * 0.62
    layer.rotation.y = i * 0.18
    group.add(layer)
  }
}

function buildGrid(group) {
  // 3x3 小方块阵列
  for (let x = -1; x <= 1; x += 1) {
    for (let z = -1; z <= 1; z += 1) {
      const cell = makeBox(0.52, (x + z) % 2 === 0 ? PURPLE : BLUE, 0.16)
      cell.position.set(x * 0.85, 0, z * 0.85)
      group.add(cell)
    }
  }
}

function buildStream(group) {
  // 环绕流动的粒子
  const count = 520
  const positions = new Float32Array(count * 3)
  for (let i = 0; i < count; i += 1) {
    const t = (i / count) * Math.PI * 2
    const r = 1.2 + Math.sin(t * 3) * 0.35
    positions[i * 3] = Math.cos(t) * r
    positions[i * 3 + 1] = Math.sin(t * 2) * 0.8
    positions[i * 3 + 2] = Math.sin(t) * r
  }
  const geo = new THREE.BufferGeometry()
  geo.setAttribute('position', new THREE.BufferAttribute(positions, 3))
  group.add(
    new THREE.Points(
      geo,
      new THREE.PointsMaterial({
        color: LIGHT,
        size: 0.045,
        transparent: true,
        opacity: 0.85,
      }),
    ),
  )
}

function buildTorus(group) {
  // 交织的线框环
  const ring1 = new THREE.Mesh(
    new THREE.TorusGeometry(1.5, 0.02, 8, 120),
    new THREE.MeshBasicMaterial({ color: PURPLE, transparent: true, opacity: 0.8 }),
  )
  ring1.rotation.x = Math.PI / 2.4
  group.add(ring1)

  const ring2 = new THREE.Mesh(
    new THREE.TorusGeometry(1.9, 0.014, 8, 120),
    new THREE.MeshBasicMaterial({ color: BLUE, transparent: true, opacity: 0.65 }),
  )
  ring2.rotation.x = -Math.PI / 3
  ring2.rotation.y = 0.5
  group.add(ring2)

  group.add(makeBox(1.0, PURPLE, 0.14))
}

const builders = {
  devices: buildDevices,
  layers: buildLayers,
  grid: buildGrid,
  stream: buildStream,
  torus: buildTorus,
}

const scenes = []

function createFeatureScene(canvas) {
  const wrap = canvas.parentElement
  const renderer = new THREE.WebGLRenderer({ canvas, antialias: true, alpha: true })
  renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2))

  const scene = new THREE.Scene()
  const camera = new THREE.PerspectiveCamera(38, 1, 0.1, 100)
  camera.position.set(0, 0, 5.6)

  const group = new THREE.Group()
  scene.add(group)
  const build = builders[canvas.dataset.scene] || buildDevices
  build(group)

  const state = { visible: false, renderer, scene, camera, group, phase: Math.random() * 6 }

  function resize() {
    const width = wrap.clientWidth
    const height = wrap.clientHeight
    renderer.setSize(width, height, false)
    camera.aspect = width / height
    camera.updateProjectionMatrix()
  }

  new ResizeObserver(resize).observe(wrap)
  resize()

  scenes.push(state)
  return state
}

document.querySelectorAll('.feature-canvas').forEach((canvas) => {
  const state = createFeatureScene(canvas)
  const observer = new IntersectionObserver(
    (entries) => {
      entries.forEach((entry) => {
        state.visible = entry.isIntersecting
      })
    },
    { threshold: 0.05 },
  )
  observer.observe(canvas.parentElement)
})

const clock = new THREE.Clock()

function animate() {
  const t = clock.getElapsedTime()
  scenes.forEach((state) => {
    if (!state.visible) {
      return
    }
    state.group.rotation.y = t * 0.35 + state.phase
    state.group.rotation.x = Math.sin(t * 0.4 + state.phase) * 0.18
    state.group.position.y = Math.sin(t * 0.7 + state.phase) * 0.12

    // 细小的节点漂移，让结构有“呼吸感”
    state.group.children.forEach((child, index) => {
      if (child.type === 'Group') {
        child.rotation.y += 0.004 * (index % 2 === 0 ? 1 : -1)
      }
    })

    state.renderer.render(state.scene, state.camera)
  })
  requestAnimationFrame(animate)
}

animate()
