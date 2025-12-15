import React, { useState, useEffect } from 'react'
import '../styles/DataConverter.css'

interface PollingInstance {
  instance_id: string
  endpoint_id: string
  label: string
  provider: string
  status: string
}

interface PollingStatus {
  config_id: string
  provider: string
  site_id: string
  tenant_id: string
  status: string
  instances: PollingInstance[]
}

interface PollingConfig {
  id: string
  provider: string
  site_id: string
  tenant_id: string
  status: string
  endpoints: EndpointConfig[]
}

interface EndpointConfig {
  instance_id: string
  endpoint_id: string
  label: string
  enabled: boolean
}

interface Recipe {
  id: string
  name: string
  description: string
  provider: string
  endpoint_id: string
  source_path: string
  field_mappings: FieldMapping[]
  static_fields: StaticField[]
  created_at: string
  updated_at: string
}

interface FieldMapping {
  from: string
  to: string
  type: 'string' | 'number' | 'boolean' | 'array' | 'object'
  transform?: string
}

interface StaticField {
  field: string
  value: string
}

interface TreeNode {
  key: string
  path: string
  value: unknown
  type: string
  children?: TreeNode[]
  expanded?: boolean
}

const DataConverter: React.FC = () => {
  const [pollingStatuses, setPollingStatuses] = useState<PollingStatus[]>([])
  const [selectedInstance, setSelectedInstance] = useState<string>('')
  const [rawData, setRawData] = useState<unknown>(null)
  const [treeData, setTreeData] = useState<TreeNode[]>([])
  const [outputFields, setOutputFields] = useState<FieldMapping[]>([])
  const [staticFields, setStaticFields] = useState<StaticField[]>([
    { field: 'schema', value: 'sensor-reading/v1' },
    { field: 'timestamp', value: '$NOW' },
  ])
  const [recipeName, setRecipeName] = useState('')
  const [recipeDescription, setRecipeDescription] = useState('')
  const [loading, setLoading] = useState(false)
  const [previewOutput, setPreviewOutput] = useState<unknown>(null)
  const [savedRecipes, setSavedRecipes] = useState<Recipe[]>([])
  const [activeTab, setActiveTab] = useState<'builder' | 'recipes'>('builder')
  const [draggedNode, setDraggedNode] = useState<TreeNode | null>(null)

  // Cargar polling statuses al montar
  useEffect(() => {
    loadPollingStatuses()
    loadSavedRecipes()
  }, [])

  // Actualizar preview cuando cambian los mappings
  useEffect(() => {
    if (rawData && outputFields.length > 0) {
      generatePreview()
    }
  }, [outputFields, staticFields, rawData])

  const loadPollingStatuses = async () => {
    try {
      const token = localStorage.getItem('token')
      const response = await fetch('http://localhost:3000/api/polling/configs', {
        headers: { Authorization: `Bearer ${token}` },
      })
      if (response.ok) {
        const data = await response.json()
        // Transform configs to the format we need (with instances)
        const configs = (data.data || []).map((config: PollingConfig) => ({
          config_id: config.id,
          provider: config.provider,
          site_id: config.site_id,
          tenant_id: config.tenant_id,
          status: config.status,
          instances: config.endpoints.map((ep: EndpointConfig) => ({
            instance_id: ep.instance_id,
            endpoint_id: ep.endpoint_id,
            label: ep.label,
            provider: config.provider,
            status: ep.enabled ? 'active' : 'stopped',
          })),
        }))
        setPollingStatuses(configs)
      }
    } catch (error) {
      console.error('Error loading polling statuses:', error)
    }
  }

  const loadSavedRecipes = async () => {
    try {
      const token = localStorage.getItem('token')
      const response = await fetch('http://localhost:3000/api/recipes', {
        headers: { Authorization: `Bearer ${token}` },
      })
      if (response.ok) {
        const data = await response.json()
        setSavedRecipes(data.recipes || [])
      }
    } catch (error) {
      console.error('Error loading recipes:', error)
    }
  }

  const loadLastResult = async (instanceId: string) => {
    setLoading(true)
    try {
      const token = localStorage.getItem('token')
      const response = await fetch(`http://localhost:3000/api/polling/last-result/${instanceId}`, {
        headers: { Authorization: `Bearer ${token}` },
      })
      if (response.ok) {
        const data = await response.json()
        setRawData(data.data)
        setTreeData(buildTree(data.data, ''))
      } else {
        console.error('No hay datos disponibles para este endpoint')
        setRawData(null)
        setTreeData([])
      }
    } catch (error) {
      console.error('Error loading last result:', error)
    } finally {
      setLoading(false)
    }
  }

  const buildTree = (obj: unknown, parentPath: string): TreeNode[] => {
    if (obj === null || obj === undefined) return []

    if (Array.isArray(obj)) {
      return obj.map((item, index) => {
        const path = parentPath ? `${parentPath}[${index}]` : `[${index}]`
        const type = typeof item
        return {
          key: `[${index}]`,
          path,
          value: item,
          type: Array.isArray(item) ? 'array' : type,
          children: type === 'object' && item !== null ? buildTree(item, path) : undefined,
          expanded: false,
        }
      })
    }

    if (typeof obj === 'object') {
      return Object.entries(obj).map(([key, value]) => {
        const path = parentPath ? `${parentPath}.${key}` : key
        const type = typeof value
        return {
          key,
          path,
          value,
          type: Array.isArray(value) ? 'array' : type,
          children: type === 'object' && value !== null ? buildTree(value, path) : undefined,
          expanded: false,
        }
      })
    }

    return []
  }

  const toggleNode = (path: string) => {
    const toggleInTree = (nodes: TreeNode[]): TreeNode[] => {
      return nodes.map((node) => {
        if (node.path === path) {
          return { ...node, expanded: !node.expanded }
        }
        if (node.children) {
          return { ...node, children: toggleInTree(node.children) }
        }
        return node
      })
    }
    setTreeData(toggleInTree(treeData))
  }

  const handleDragStart = (node: TreeNode) => {
    setDraggedNode(node)
  }

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault()
  }

  const handleDropOnOutput = (e: React.DragEvent) => {
    e.preventDefault()
    if (draggedNode && draggedNode.type !== 'object' && draggedNode.type !== 'array') {
      // Agregar campo al output
      const newMapping: FieldMapping = {
        from: draggedNode.path,
        to: draggedNode.key,
        type: draggedNode.type as FieldMapping['type'],
      }
      setOutputFields([...outputFields, newMapping])
    }
    setDraggedNode(null)
  }

  const updateOutputField = (index: number, field: 'to' | 'type', value: string) => {
    const updated = [...outputFields]
    updated[index] = { ...updated[index], [field]: value }
    setOutputFields(updated)
  }

  const removeOutputField = (index: number) => {
    setOutputFields(outputFields.filter((_, i) => i !== index))
  }

  const addStaticField = () => {
    setStaticFields([...staticFields, { field: '', value: '' }])
  }

  const updateStaticField = (index: number, key: 'field' | 'value', value: string) => {
    const updated = [...staticFields]
    updated[index] = { ...updated[index], [key]: value }
    setStaticFields(updated)
  }

  const removeStaticField = (index: number) => {
    setStaticFields(staticFields.filter((_, i) => i !== index))
  }

  const getValueByPath = (obj: unknown, path: string): unknown => {
    if (!path) return obj

    const parts = path.replace(/\[(\d+)\]/g, '.$1').split('.')
    let current: unknown = obj

    for (const part of parts) {
      if (current === null || current === undefined) return undefined
      if (typeof current === 'object') {
        current = (current as Record<string, unknown>)[part]
      } else {
        return undefined
      }
    }

    return current
  }

  const generatePreview = () => {
    if (!rawData) return

    const output: Record<string, unknown> = {}

    // Campos estáticos
    for (const sf of staticFields) {
      if (sf.field) {
        if (sf.value === '$NOW') {
          output[sf.field] = new Date().toISOString()
        } else {
          output[sf.field] = sf.value
        }
      }
    }

    // Campos mapeados
    for (const mapping of outputFields) {
      const value = getValueByPath(rawData, mapping.from)
      if (value !== undefined) {
        // Soportar paths anidados en el output (ej: "readings.oxygen")
        const parts = mapping.to.split('.')
        let current = output
        for (let i = 0; i < parts.length - 1; i++) {
          if (!current[parts[i]]) {
            current[parts[i]] = {}
          }
          current = current[parts[i]] as Record<string, unknown>
        }
        current[parts[parts.length - 1]] = value
      }
    }

    setPreviewOutput(output)
  }

  const saveRecipe = async () => {
    if (!recipeName || outputFields.length === 0) {
      alert('Por favor ingresa un nombre y al menos un campo mapeado')
      return
    }

    // Encontrar el endpoint seleccionado
    let provider = ''
    let endpointId = ''
    for (const status of pollingStatuses) {
      const instance = status.instances.find((i) => i.instance_id === selectedInstance)
      if (instance) {
        provider = status.provider
        endpointId = instance.endpoint_id
        break
      }
    }

    const recipe = {
      name: recipeName,
      description: recipeDescription,
      provider,
      endpoint_id: endpointId,
      instance_id: selectedInstance,
      source_path: '', // Por ahora root
      field_mappings: outputFields,
      static_fields: staticFields,
    }

    try {
      const token = localStorage.getItem('token')
      const response = await fetch('http://localhost:3000/api/recipes', {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(recipe),
      })

      if (response.ok) {
        alert('Receta guardada exitosamente!')
        loadSavedRecipes()
        setRecipeName('')
        setRecipeDescription('')
      } else {
        const error = await response.json()
        alert(`Error: ${error.error || 'No se pudo guardar la receta'}`)
      }
    } catch (error) {
      console.error('Error saving recipe:', error)
      alert('Error al guardar la receta')
    }
  }

  const loadRecipe = (recipe: Recipe) => {
    setRecipeName(recipe.name)
    setRecipeDescription(recipe.description)
    setOutputFields(recipe.field_mappings)
    setStaticFields(recipe.static_fields)
    setActiveTab('builder')
  }

  const deleteRecipe = async (recipeId: string) => {
    if (!confirm('¿Estás seguro de eliminar esta receta?')) return

    try {
      const token = localStorage.getItem('token')
      const response = await fetch(`http://localhost:3000/api/recipes/${recipeId}`, {
        method: 'DELETE',
        headers: { Authorization: `Bearer ${token}` },
      })

      if (response.ok) {
        loadSavedRecipes()
      }
    } catch (error) {
      console.error('Error deleting recipe:', error)
    }
  }

  const renderTreeNode = (node: TreeNode, depth: number = 0): React.ReactNode => {
    const hasChildren = node.children && node.children.length > 0
    const isExpandable = node.type === 'object' || node.type === 'array'
    const isDraggable = !isExpandable

    return (
      <div key={node.path} className="tree-node" style={{ marginLeft: depth * 16 }}>
        <div
          className={`tree-node-content ${isDraggable ? 'draggable' : ''}`}
          draggable={isDraggable}
          onDragStart={() => isDraggable && handleDragStart(node)}
        >
          {isExpandable && (
            <span className="tree-toggle" onClick={() => toggleNode(node.path)}>
              {node.expanded ? '▼' : '▶'}
            </span>
          )}
          {!isExpandable && <span className="tree-toggle-placeholder">•</span>}

          <span className="tree-key">{node.key}</span>
          <span className={`tree-type type-${node.type}`}>{node.type}</span>

          {!isExpandable && (
            <span className="tree-value">
              {JSON.stringify(node.value).substring(0, 50)}
              {JSON.stringify(node.value).length > 50 ? '...' : ''}
            </span>
          )}
          {isExpandable && (
            <span className="tree-count">
              {node.type === 'array'
                ? `[${(node.value as unknown[]).length}]`
                : `{${Object.keys(node.value as object).length}}`}
            </span>
          )}
        </div>

        {node.expanded && hasChildren && (
          <div className="tree-children">
            {node.children!.map((child) => renderTreeNode(child, depth + 1))}
          </div>
        )}
      </div>
    )
  }

  // Flatten all instances from all polling configs
  const allInstances = pollingStatuses.flatMap((status) =>
    status.instances.map((instance) => ({
      ...instance,
      provider: status.provider,
      configId: status.config_id,
    }))
  )

  return (
    <div className="data-converter">
      <div className="page-header">
        <h1>🔄 Data Converter</h1>
        <p>Transforma datos de endpoints a formatos estandarizados</p>
      </div>

      <div className="tabs">
        <button
          className={`tab ${activeTab === 'builder' ? 'active' : ''}`}
          onClick={() => setActiveTab('builder')}
        >
          🛠️ Constructor
        </button>
        <button
          className={`tab ${activeTab === 'recipes' ? 'active' : ''}`}
          onClick={() => setActiveTab('recipes')}
        >
          📋 Recetas Guardadas ({savedRecipes.length})
        </button>
      </div>

      {activeTab === 'builder' && (
        <div className="builder-content">
          {/* Selector de endpoint */}
          <div className="endpoint-selector">
            <label>Seleccionar Endpoint Activo:</label>
            <select
              value={selectedInstance}
              onChange={(e) => {
                setSelectedInstance(e.target.value)
                if (e.target.value) {
                  loadLastResult(e.target.value)
                }
              }}
            >
              <option value="">-- Seleccionar --</option>
              {allInstances.map((instance) => (
                <option key={instance.instance_id} value={instance.instance_id}>
                  [{instance.provider}] {instance.label}
                </option>
              ))}
            </select>
            {loading && <span className="loading-indicator">Cargando datos...</span>}
          </div>

          {rawData && (
            <div className="converter-workspace">
              {/* Panel izquierdo: JSON Tree */}
              <div className="panel input-panel">
                <div className="panel-header">
                  <h3>📥 Datos de Entrada (Raw)</h3>
                  <span className="hint">Arrastra campos al panel de salida</span>
                </div>
                <div className="panel-content tree-container">
                  {treeData.map((node) => renderTreeNode(node))}
                </div>
              </div>

              {/* Panel central: Output mapping */}
              <div
                className="panel output-panel"
                onDragOver={handleDragOver}
                onDrop={handleDropOnOutput}
              >
                <div className="panel-header">
                  <h3>📤 Estructura de Salida</h3>
                  <span className="hint">Suelta campos aquí</span>
                </div>
                <div className="panel-content">
                  {/* Campos estáticos */}
                  <div className="fields-section">
                    <h4>Campos Estáticos</h4>
                    {staticFields.map((sf, index) => (
                      <div key={index} className="field-row static-field">
                        <input
                          type="text"
                          value={sf.field}
                          onChange={(e) => updateStaticField(index, 'field', e.target.value)}
                          placeholder="nombre"
                        />
                        <span>=</span>
                        <input
                          type="text"
                          value={sf.value}
                          onChange={(e) => updateStaticField(index, 'value', e.target.value)}
                          placeholder="valor ($NOW para timestamp)"
                        />
                        <button className="btn-remove" onClick={() => removeStaticField(index)}>
                          ×
                        </button>
                      </div>
                    ))}
                    <button className="btn-add" onClick={addStaticField}>
                      + Agregar campo estático
                    </button>
                  </div>

                  {/* Campos mapeados */}
                  <div className="fields-section">
                    <h4>Campos Mapeados</h4>
                    {outputFields.length === 0 && (
                      <div className="drop-zone">Arrastra campos desde el panel izquierdo</div>
                    )}
                    {outputFields.map((mapping, index) => (
                      <div key={index} className="field-row mapped-field">
                        <span className="field-from" title={mapping.from}>
                          {mapping.from.split('.').pop()}
                        </span>
                        <span className="arrow">→</span>
                        <input
                          type="text"
                          value={mapping.to}
                          onChange={(e) => updateOutputField(index, 'to', e.target.value)}
                          placeholder="nombre destino"
                        />
                        <select
                          value={mapping.type}
                          onChange={(e) => updateOutputField(index, 'type', e.target.value)}
                        >
                          <option value="string">string</option>
                          <option value="number">number</option>
                          <option value="boolean">boolean</option>
                        </select>
                        <button className="btn-remove" onClick={() => removeOutputField(index)}>
                          ×
                        </button>
                      </div>
                    ))}
                  </div>
                </div>
              </div>

              {/* Panel derecho: Preview */}
              <div className="panel preview-panel">
                <div className="panel-header">
                  <h3>👁️ Preview</h3>
                  <button className="btn-refresh" onClick={generatePreview}>
                    🔄 Actualizar
                  </button>
                </div>
                <div className="panel-content">
                  <pre className="preview-json">
                    {previewOutput
                      ? JSON.stringify(previewOutput, null, 2)
                      : '// El resultado aparecerá aquí'}
                  </pre>
                </div>
              </div>
            </div>
          )}

          {/* Guardar receta */}
          {outputFields.length > 0 && (
            <div className="save-recipe-section">
              <h3>💾 Guardar Receta</h3>
              <div className="save-form">
                <input
                  type="text"
                  value={recipeName}
                  onChange={(e) => setRecipeName(e.target.value)}
                  placeholder="Nombre de la receta"
                  className="recipe-name-input"
                />
                <input
                  type="text"
                  value={recipeDescription}
                  onChange={(e) => setRecipeDescription(e.target.value)}
                  placeholder="Descripción (opcional)"
                  className="recipe-desc-input"
                />
                <button className="btn-save" onClick={saveRecipe}>
                  💾 Guardar Receta
                </button>
              </div>
            </div>
          )}
        </div>
      )}

      {activeTab === 'recipes' && (
        <div className="recipes-list">
          {savedRecipes.length === 0 ? (
            <div className="empty-state">
              <p>No hay recetas guardadas</p>
              <p>Crea una en el Constructor</p>
            </div>
          ) : (
            <div className="recipes-grid">
              {savedRecipes.map((recipe) => (
                <div key={recipe.id} className="recipe-card">
                  <div className="recipe-header">
                    <h3>{recipe.name}</h3>
                    <span className="recipe-provider">{recipe.provider}</span>
                  </div>
                  <p className="recipe-description">{recipe.description || 'Sin descripción'}</p>
                  <div className="recipe-meta">
                    <span>📊 {recipe.field_mappings.length} campos</span>
                    <span>🎯 {recipe.endpoint_id}</span>
                  </div>
                  <div className="recipe-actions">
                    <button onClick={() => loadRecipe(recipe)}>✏️ Editar</button>
                    <button onClick={() => deleteRecipe(recipe.id)} className="btn-danger">
                      🗑️ Eliminar
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

export default DataConverter
