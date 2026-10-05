import React, { useState, useEffect } from 'react'
import {
  getHoldBorrowList,
  submitHoldBorrow,
  rejectHoldBorrow,
  updateHoldBorrow,
  getAssets,
} from './services.js'

export default function HoldBorrowPage({ showToast, onBack, onGoApprovals }) {
  const [allRecords, setAllRecords] = useState([])
  const [loading, setLoading] = useState(true)
  const [activeTab, setActiveTab] = useState('held') // 'held' | 'submitted' | 'all'
  const [processingId, setProcessingId] = useState(null)

  // Edit / Confirmation Modal State
  const [editModalOpen, setEditModalOpen] = useState(false)
  const [modalMode, setModalMode] = useState('edit') // 'edit' | 'confirm'
  const [editingRec, setEditingRec] = useState(null)
  const [editForm, setEditForm] = useState({
    borrower_name: '',
    group_name: '',
    notes: '',
    items: [],
  })
  const [formError, setFormError] = useState('')
  const [isSaving, setIsSaving] = useState(false)
  const [isSubmitting, setIsSubmitting] = useState(false)

  // Available Assets catalogue for adding items
  const [availableAssets, setAvailableAssets] = useState([])
  const [loadingAssets, setLoadingAssets] = useState(false)
  const [showAddAssetRow, setShowAddAssetRow] = useState(false)
  const [selectedNewAssetId, setSelectedNewAssetId] = useState('')
  const [newAssetDuration, setNewAssetDuration] = useState(14)
  const [assetSearchQuery, setAssetSearchQuery] = useState('')

  const loadData = async () => {
    setLoading(true)
    try {
      const res = await getHoldBorrowList('all')
      setAllRecords(res.data || [])
    } catch (err) {
      showToast('Gagal memuat antrean hold borrow: ' + err.message, 'warning')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadData()
  }, [])

  const heldCount = allRecords.filter((r) => r.status === 'held').length
  const submittedCount = allRecords.filter((r) => r.status === 'submitted').length
  const allCount = allRecords.length

  const filteredRecords = allRecords.filter((r) => {
    if (activeTab === 'all') return true
    return r.status === activeTab
  })

  // Open Edit / Confirm Modal
  const openEditModal = (rec, mode = 'edit') => {
    setEditingRec(rec)
    setModalMode(mode)
    setEditForm({
      borrower_name: rec.borrower_name || '',
      group_name: rec.group_name || 'qtera mandiri',
      notes: rec.notes || '',
      items: (rec.items || []).map((it) => ({
        asset_id: it.asset_id,
        asset_name: it.asset_name || it.asset_id,
        category: it.category || '',
        duration_days: parseInt(it.duration_days, 10) || 14,
      })),
    })
    setFormError('')
    setShowAddAssetRow(false)
    setSelectedNewAssetId('')
    setNewAssetDuration(14)
    setAssetSearchQuery('')
    setEditModalOpen(true)

    // Preload available assets if not loaded yet
    if (availableAssets.length === 0) {
      setLoadingAssets(true)
      getAssets({ pageSize: 200 })
        .then((res) => {
          setAvailableAssets(res.data || [])
        })
        .catch(() => {})
        .finally(() => {
          setLoadingAssets(false)
        })
    }
  }

  const closeEditModal = () => {
    if (isSaving || isSubmitting) return
    setEditModalOpen(false)
    setEditingRec(null)
  }

  // Handle duration adjustment in modal
  const handleItemDurationChange = (index, deltaOrValue, isDelta = false) => {
    setEditForm((prev) => {
      const updated = [...prev.items]
      let newDuration = isDelta
        ? (parseInt(updated[index].duration_days, 10) || 14) + deltaOrValue
        : parseInt(deltaOrValue, 10)

      if (isNaN(newDuration) || newDuration < 1) {
        newDuration = 1
      }
      updated[index] = { ...updated[index], duration_days: newDuration }
      return { ...prev, items: updated }
    })
  }

  // Handle remove item in modal
  const handleRemoveItem = (index) => {
    if (editForm.items.length <= 1) {
      setFormError('Minimal 1 aset harus ada dalam permohonan peminjaman.')
      return
    }
    setFormError('')
    setEditForm((prev) => ({
      ...prev,
      items: prev.items.filter((_, i) => i !== index),
    }))
  }

  // Handle add asset in modal
  const handleAddAssetToForm = () => {
    if (!selectedNewAssetId) {
      setFormError('Pilih aset yang ingin ditambahkan.')
      return
    }
    // Check if duplicate
    const exists = editForm.items.some((it) => it.asset_id === selectedNewAssetId)
    if (exists) {
      setFormError('Aset ini sudah ada di dalam daftar pinjaman.')
      return
    }

    const assetObj = availableAssets.find((a) => a.asset_id === selectedNewAssetId)
    const newItem = {
      asset_id: selectedNewAssetId,
      asset_name: assetObj?.name || selectedNewAssetId,
      category: assetObj?.category || 'General',
      duration_days: parseInt(newAssetDuration, 10) || 14,
    }

    setEditForm((prev) => ({
      ...prev,
      items: [...prev.items, newItem],
    }))
    setSelectedNewAssetId('')
    setShowAddAssetRow(false)
    setFormError('')
  }

  // Save edits to hold_borrow (keep status as held)
  const handleSaveEdit = async () => {
    if (!editForm.borrower_name.trim()) {
      setFormError('Nama peminjam wajib diisi.')
      return
    }
    if (editForm.items.length === 0) {
      setFormError('Minimal 1 aset harus dipinjam.')
      return
    }

    setIsSaving(true)
    setFormError('')
    try {
      await updateHoldBorrow(editingRec.id, {
        borrower_name: editForm.borrower_name.trim(),
        group_name: editForm.group_name.trim() || (editingRec?.group_name || ''),
        notes: editForm.notes.trim(),
        items: editForm.items.map((it) => ({
          asset_id: it.asset_id,
          asset_name: it.asset_name,
          category: it.category,
          duration_days: parseInt(it.duration_days, 10) || 14,
        })),
      })
      showToast(`Data permohonan hold ${editingRec.hold_code} berhasil diperbarui!`, 'success')
      closeEditModal()
      loadData()
    } catch (err) {
      setFormError('Gagal menyimpan perubahan: ' + err.message)
    } finally {
      setIsSaving(false)
    }
  }

  // Save edits & Submit immediately as official transaction
  const handleConfirmAndSubmit = async () => {
    if (!editForm.borrower_name.trim()) {
      setFormError('Nama peminjam wajib diisi sebelum konfirmasi.')
      return
    }
    if (editForm.items.length === 0) {
      setFormError('Minimal 1 aset harus dipinjam.')
      return
    }

    setIsSubmitting(true)
    setFormError('')
    try {
      const payload = {
        borrower_name: editForm.borrower_name.trim(),
        group_name: editForm.group_name.trim() || (editingRec?.group_name || ''),
        notes: editForm.notes.trim(),
        items: editForm.items.map((it) => ({
          asset_id: it.asset_id,
          asset_name: it.asset_name,
          category: it.category,
          duration_days: parseInt(it.duration_days, 10) || 14,
        })),
      }
      const res = await submitHoldBorrow(editingRec.id, payload)
      showToast(`Berhasil! Permohonan telah dipindahkan ke daftar Approval Peminjaman (${res.transaction_code}).`, 'success')
      closeEditModal()
      loadData()
    } catch (err) {
      setFormError('Gagal submit permohonan: ' + err.message)
    } finally {
      setIsSubmitting(false)
    }
  }

  const handleReject = async (rec) => {
    if (!window.confirm(`Batalkan dan hapus permohonan hold ${rec.hold_code}?`)) {
      return
    }
    setProcessingId(rec.id)
    try {
      await rejectHoldBorrow(rec.id)
      showToast(`Permohonan ${rec.hold_code} berhasil ditolak dan dihapus.`, 'info')
      loadData()
    } catch (err) {
      showToast('Gagal menghapus permohonan: ' + err.message, 'danger')
    } finally {
      setProcessingId(null)
    }
  }

  const formatDate = (dateStr) => {
    if (!dateStr) return '-'
    try {
      const d = new Date(dateStr)
      return d.toLocaleString('id-ID', {
        day: '2-digit',
        month: 'short',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
      })
    } catch {
      return dateStr
    }
  }

  const getStatusBadge = (status) => {
    switch (status?.toLowerCase()) {
      case 'held':
        return (
          <span
            className="badge badge-warning"
            style={{
              backgroundColor: '#fef3c7',
              color: '#b45309',
              padding: '4px 10px',
              borderRadius: '12px',
              fontWeight: '600',
              fontSize: '12px',
            }}
          >
            🟡 Menunggu Submit
          </span>
        )
      case 'submitted':
        return (
          <span
            className="badge badge-success"
            style={{
              backgroundColor: '#dcfce7',
              color: '#15803d',
              padding: '4px 10px',
              borderRadius: '12px',
              fontWeight: '600',
              fontSize: '12px',
            }}
          >
            🟢 Sudah Di-Submit
          </span>
        )
      case 'rejected':
        return (
          <span
            className="badge badge-danger"
            style={{
              backgroundColor: '#fee2e2',
              color: '#b91c1c',
              padding: '4px 10px',
              borderRadius: '12px',
              fontWeight: '600',
              fontSize: '12px',
            }}
          >
            🔴 Dibatalkan
          </span>
        )
      default:
        return <span className="badge badge-neutral">{status}</span>
    }
  }

  // Filter available assets for dropdown
  const filteredAssetOptions = availableAssets.filter((a) => {
    // Exclude if already in items
    const alreadySelected = editForm.items.some((it) => it.asset_id === a.asset_id)
    if (alreadySelected) return false
    if (!assetSearchQuery.trim()) return true
    const q = assetSearchQuery.toLowerCase()
    return (
      a.asset_id?.toLowerCase().includes(q) ||
      a.name?.toLowerCase().includes(q) ||
      a.category?.toLowerCase().includes(q) ||
      a.brand?.toLowerCase().includes(q)
    )
  })

  return (
    <div className="borrow-approval-container" style={{ padding: '24px', maxWidth: '1100px', margin: '0 auto' }}>
      {/* Page Header */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px', flexWrap: 'wrap', gap: '12px' }}>
        <div>
          <button className="btn btn-ghost btn-sm" onClick={onBack} style={{ marginBottom: '8px' }}>
            <i className="ph ph-arrow-left" /> Kembali ke Aset
          </button>
          <h2 style={{ fontSize: '24px', fontWeight: '700', color: '#111827', margin: 0 }}>
            ⏳ Antrean Hold Peminjaman Aset (AI Borrow Hold)
          </h2>
          <p style={{ color: '#6b7280', fontSize: '14px', marginTop: '4px' }}>
            Halaman khusus penampungan permohonan pinjam oleh AI. Anda dapat mengedit peminjam, aset, dan durasi sebelum konfirmasi submit transaksi.
          </p>
        </div>

        <div style={{ display: 'flex', gap: '8px' }}>
          <button className="btn btn-outline btn-sm" onClick={loadData} disabled={loading}>
            <i className="ph ph-arrows-clockwise" /> Refresh
          </button>
          {onGoApprovals && (
            <button className="btn btn-primary btn-sm" onClick={onGoApprovals} style={{ backgroundColor: '#4f46e5', borderColor: '#4f46e5' }}>
              <i className="ph ph-stamp" /> Buka Halaman Approvals →
            </button>
          )}
        </div>
      </div>

      {/* Info Notice Banner */}
      <div
        style={{
          backgroundColor: '#eff6ff',
          border: '1px solid #bfdbfe',
          borderRadius: '10px',
          padding: '16px',
          marginBottom: '20px',
          display: 'flex',
          gap: '14px',
          alignItems: 'flex-start',
        }}
      >
        <i className="ph ph-info" style={{ fontSize: '24px', color: '#2563eb', flexShrink: 0, marginTop: '2px' }} />
        <div>
          <strong style={{ color: '#1e40af', fontSize: '15px' }}>Fitur Edit Data Hold & Konfirmasi User</strong>
          <p style={{ color: '#1e3a8a', fontSize: '13px', margin: '4px 0 0', lineHeight: '1.5' }}>
            • <strong>Hold Borrow (Halaman Ini)</strong>: Menampung permintaan pinjam dari AI. Anda dapat mengklik tombol <strong>"Edit Data"</strong> atau <strong>"Konfirmasi & Submit"</strong> untuk memeriksa, menambah/menghapus barang, menyesuaikan durasi pinjam, atau mengganti nama peminjam sebelum diproses ke sistem.<br />
            • <strong>Approvals Page</strong>: Halaman untuk manager menyetujui transaksi resmi yang telah diajukan (TRX-BRW-XXX).
          </p>
        </div>
      </div>

      {/* Tab Filter */}
      <div style={{ display: 'flex', gap: '8px', marginBottom: '20px', borderBottom: '1px solid #e5e7eb', paddingBottom: '12px', flexWrap: 'wrap' }}>
        <button
          className={`btn btn-sm ${activeTab === 'held' ? 'btn-primary' : 'btn-ghost'}`}
          onClick={() => setActiveTab('held')}
          style={activeTab === 'held' ? { backgroundColor: '#2563eb' } : {}}
        >
          🟡 Menunggu Submit ({heldCount})
        </button>
        <button
          className={`btn btn-sm ${activeTab === 'submitted' ? 'btn-primary' : 'btn-ghost'}`}
          onClick={() => setActiveTab('submitted')}
          style={activeTab === 'submitted' ? { backgroundColor: '#16a34a', borderColor: '#16a34a' } : {}}
        >
          🟢 Sudah Di-Submit ({submittedCount})
        </button>
        <button
          className={`btn btn-sm ${activeTab === 'all' ? 'btn-primary' : 'btn-ghost'}`}
          onClick={() => setActiveTab('all')}
          style={activeTab === 'all' ? { backgroundColor: '#475569', borderColor: '#475569' } : {}}
        >
          📑 Semua ({allCount})
        </button>
      </div>

      {/* List of Held Requests */}
      {loading ? (
        <div style={{ textAlign: 'center', padding: '48px 0', color: '#6b7280' }}>
          <i className="ph ph-spinner ph-spin" style={{ fontSize: '32px', color: '#2563eb', marginBottom: '12px' }} />
          <div>Memuat data antrean hold borrow...</div>
        </div>
      ) : filteredRecords.length === 0 ? (
        <div style={{ textAlign: 'center', padding: '56px 20px', backgroundColor: '#f9fafb', borderRadius: '12px', border: '1px dashed #d1d5db' }}>
          <i className="ph ph-tray" style={{ fontSize: '48px', color: '#9ca3af', marginBottom: '12px' }} />
          <h4 style={{ fontSize: '16px', fontWeight: '600', color: '#374151', margin: '0 0 6px' }}>
            {activeTab === 'held' && 'Tidak ada permohonan yang sedang menunggu submit.'}
            {activeTab === 'submitted' && 'Belum ada permohonan yang sudah di-submit.'}
            {activeTab === 'all' && 'Belum ada riwayat permohonan hold.'}
          </h4>
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
          {filteredRecords.map((rec) => {
            const isBusy = processingId === rec.id

            return (
              <div
                key={rec.id}
                style={{
                  backgroundColor: '#ffffff',
                  border: '1px solid #e5e7eb',
                  borderRadius: '12px',
                  padding: '20px',
                  boxShadow: '0 1px 3px rgba(0,0,0,0.06)',
                }}
              >
                {/* Header Card */}
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', paddingBottom: '12px', borderBottom: '1px solid #f3f4f6', marginBottom: '14px' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                    <span style={{ fontSize: '18px', fontWeight: '700', fontFamily: 'monospace', color: '#111827' }}>
                      {rec.hold_code}
                    </span>
                    {getStatusBadge(rec.status)}
                  </div>
                  <span style={{ fontSize: '13px', color: '#6b7280' }}>
                    {formatDate(rec.created_at)}
                  </span>
                </div>

                {/* Metadata Details */}
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: '12px', marginBottom: '16px', fontSize: '13px' }}>
                  <div>
                    <span style={{ color: '#6b7280', display: 'block', fontSize: '12px' }}>Peminjam</span>
                    <strong style={{ color: '#111827' }}>👤 {rec.borrower_name}</strong>
                    <span style={{ color: '#9ca3af', fontSize: '11px', display: 'block' }}>ID: {rec.borrower_id || '-'}</span>
                  </div>

                  <div>
                    <span style={{ color: '#6b7280', display: 'block', fontSize: '12px' }}>Manager / Pemohon (Login)</span>
                    <strong style={{ color: '#111827' }}>🛡️ {rec.manager_name}</strong>
                    <span style={{ color: '#9ca3af', fontSize: '11px', display: 'block' }}>ID: {rec.manager_id || '-'}</span>
                  </div>

                  <div>
                    <span style={{ color: '#6b7280', display: 'block', fontSize: '12px' }}>Groups / Divisi</span>
                    <strong style={{ color: '#111827' }}>🏢 {rec.group_name || '—'}</strong>
                  </div>

                  <div>
                    <span style={{ color: '#6b7280', display: 'block', fontSize: '12px' }}>Catatan AI</span>
                    <span style={{ color: '#4b5563', fontStyle: 'italic' }}>{rec.notes || 'Dibuat otomatis oleh AI'}</span>
                  </div>
                </div>

                {/* Items List */}
                <div style={{ backgroundColor: '#f9fafb', borderRadius: '8px', padding: '14px', marginBottom: '16px' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '8px' }}>
                    <div style={{ fontWeight: '600', fontSize: '13px', color: '#374151' }}>
                      Daftar Aset yang Dipinjam ({rec.items?.length || 0}):
                    </div>
                    {rec.status === 'held' && (
                      <button
                        type="button"
                        onClick={() => openEditModal(rec, 'edit')}
                        style={{
                          background: 'none',
                          border: 'none',
                          color: '#2563eb',
                          fontSize: '12px',
                          cursor: 'pointer',
                          fontWeight: '600',
                          display: 'flex',
                          alignItems: 'center',
                          gap: '4px',
                        }}
                      >
                        <i className="ph ph-pencil-simple" /> Edit Aset / Durasi
                      </button>
                    )}
                  </div>
                  {rec.items && rec.items.length > 0 ? (
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                      {rec.items.map((it, idx) => (
                        <div
                          key={idx}
                          style={{
                            display: 'flex',
                            justifyContent: 'space-between',
                            alignItems: 'center',
                            backgroundColor: '#ffffff',
                            border: '1px solid #e5e7eb',
                            padding: '8px 12px',
                            borderRadius: '6px',
                          }}
                        >
                          <div>
                            <strong style={{ fontSize: '13px', color: '#111827' }}>{it.asset_name || it.asset_id}</strong>
                            <div style={{ fontSize: '11px', color: '#6b7280' }}>
                              ID: <code style={{ backgroundColor: '#f3f4f6', padding: '1px 4px', borderRadius: '4px' }}>{it.asset_id}</code> {it.category ? `• ${it.category}` : ''}
                            </div>
                          </div>
                          <span
                            style={{
                              backgroundColor: '#dbeafe',
                              color: '#1d4ed8',
                              padding: '3px 8px',
                              borderRadius: '6px',
                              fontSize: '12px',
                              fontWeight: '600',
                            }}
                          >
                            ⏳ {it.duration_days || 14} Hari
                          </span>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <span style={{ color: '#9ca3af', fontSize: '12px' }}>Tidak ada aset tercatat.</span>
                  )}
                </div>

                {/* Actions */}
                {rec.status === 'held' && (
                  <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px', flexWrap: 'wrap' }}>
                    <button
                      className="btn btn-outline btn-sm"
                      onClick={() => handleReject(rec)}
                      disabled={isBusy}
                      style={{ color: '#dc2626', borderColor: '#fca5a5' }}
                    >
                      <i className="ph ph-x" /> Batalkan
                    </button>

                    <button
                      className="btn btn-outline btn-sm"
                      onClick={() => openEditModal(rec, 'edit')}
                      disabled={isBusy}
                      style={{ color: '#2563eb', borderColor: '#bfdbfe', backgroundColor: '#f0f7ff' }}
                    >
                      <i className="ph ph-pencil-simple" /> Edit Data
                    </button>

                    <button
                      className="btn btn-primary btn-sm"
                      onClick={() => openEditModal(rec, 'confirm')}
                      disabled={isBusy}
                      style={{ backgroundColor: '#16a34a', borderColor: '#16a34a', minWidth: '170px' }}
                    >
                      {isBusy ? (
                        <i className="ph ph-spinner ph-spin" />
                      ) : (
                        <>
                          <i className="ph ph-check-circle" /> Konfirmasi & Submit
                        </>
                      )}
                    </button>
                  </div>
                )}
              </div>
            )
          })}
        </div>
      )}

      {/* Edit & Confirmation Modal */}
      {editModalOpen && editingRec && (
        <div
          style={{
            position: 'fixed',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            backgroundColor: 'rgba(15, 23, 42, 0.65)',
            backdropFilter: 'blur(4px)',
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
            zIndex: 1000,
            padding: '16px',
          }}
          onClick={(e) => {
            if (e.target === e.currentTarget) closeEditModal()
          }}
        >
          <div
            style={{
              backgroundColor: '#ffffff',
              borderRadius: '16px',
              maxWidth: '680px',
              width: '100%',
              maxHeight: '92vh',
              overflowY: 'auto',
              padding: '24px',
              boxShadow: '0 20px 25px -5px rgba(0, 0, 0, 0.15), 0 10px 10px -5px rgba(0, 0, 0, 0.04)',
            }}
          >
            {/* Modal Header */}
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px', borderBottom: '1px solid #f1f5f9', paddingBottom: '12px' }}>
              <div>
                <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                  <h3 style={{ fontSize: '18px', fontWeight: '700', color: '#0f172a', margin: 0 }}>
                    {modalMode === 'confirm' ? '✅ Konfirmasi & Submit Peminjaman' : '✏️ Edit Permohonan Hold Aset'}
                  </h3>
                  <span style={{ fontSize: '13px', fontFamily: 'monospace', backgroundColor: '#e2e8f0', padding: '2px 8px', borderRadius: '6px', fontWeight: '600' }}>
                    {editingRec.hold_code}
                  </span>
                </div>
                <p style={{ fontSize: '13px', color: '#64748b', margin: '4px 0 0 0' }}>
                  {modalMode === 'confirm'
                    ? 'Periksa dan sesuaikan data peminjaman di bawah ini sebelum konfirmasi final ke transaksi resmi.'
                    : 'Ubah informasi peminjam, grup, catatan, atau rincian aset yang ditahan.'}
                </p>
              </div>
              <button
                type="button"
                onClick={closeEditModal}
                disabled={isSaving || isSubmitting}
                style={{
                  background: 'none',
                  border: 'none',
                  cursor: 'pointer',
                  fontSize: '20px',
                  color: '#94a3b8',
                  padding: '4px',
                }}
              >
                <i className="ph ph-x" />
              </button>
            </div>

            {/* Notification Banner */}
            {modalMode === 'confirm' ? (
              <div
                style={{
                  backgroundColor: '#f0fdf4',
                  border: '1px solid #bbf7d0',
                  borderRadius: '8px',
                  padding: '12px 14px',
                  marginBottom: '16px',
                  display: 'flex',
                  gap: '10px',
                  alignItems: 'flex-start',
                }}
              >
                <i className="ph ph-check-circle" style={{ fontSize: '20px', color: '#16a34a', flexShrink: 0, marginTop: '1px' }} />
                <div style={{ fontSize: '13px', color: '#166534', lineHeight: '1.4' }}>
                  <strong>Mode Konfirmasi:</strong> Data di bawah siap dijadikan transaksi resmi. Jika ada durasi atau nama yang keliru dari AI, silakan edit langsung di form ini lalu klik <em>"Konfirmasi & Submit Resmi"</em>.
                </div>
              </div>
            ) : (
              <div
                style={{
                  backgroundColor: '#eff6ff',
                  border: '1px solid #bfdbfe',
                  borderRadius: '8px',
                  padding: '12px 14px',
                  marginBottom: '16px',
                  display: 'flex',
                  gap: '10px',
                  alignItems: 'flex-start',
                }}
              >
                <i className="ph ph-pencil-simple" style={{ fontSize: '20px', color: '#2563eb', flexShrink: 0, marginTop: '1px' }} />
                <div style={{ fontSize: '13px', color: '#1e40af', lineHeight: '1.4' }}>
                  <strong>Mode Edit:</strong> Anda dapat mengedit rincian data lalu menekan <em>"Simpan Perubahan"</em> untuk tetap menahan permohonan, atau langsung <em>"Konfirmasi & Submit"</em>.
                </div>
              </div>
            )}

            {/* Error Message */}
            {formError && (
              <div
                style={{
                  backgroundColor: '#fef2f2',
                  border: '1px solid #fecaca',
                  color: '#dc2626',
                  borderRadius: '8px',
                  padding: '10px 14px',
                  fontSize: '13px',
                  marginBottom: '16px',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '8px',
                }}
              >
                <i className="ph ph-warning-circle" style={{ fontSize: '18px' }} />
                {formError}
              </div>
            )}

            {/* Form Fields */}
            <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
              {/* Row 1: Borrower Name & Group Name */}
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))', gap: '14px' }}>
                <div>
                  <label style={{ display: 'block', fontSize: '13px', fontWeight: '600', color: '#334155', marginBottom: '6px' }}>
                    👤 Nama Peminjam <span style={{ color: '#ef4444' }}>*</span>
                  </label>
                  <input
                    type="text"
                    value={editForm.borrower_name}
                    onChange={(e) => setEditForm({ ...editForm, borrower_name: e.target.value })}
                    placeholder="Contoh: Dan, Budi Santoso"
                    style={{
                      width: '100%',
                      padding: '9px 12px',
                      borderRadius: '8px',
                      border: '1px solid #cbd5e1',
                      fontSize: '14px',
                      backgroundColor: '#ffffff',
                    }}
                  />
                  <span style={{ fontSize: '11px', color: '#94a3b8', marginTop: '3px', display: 'block' }}>
                    Sistem akan otomatis mencocokkan user ID peminjam jika terdaftar.
                  </span>
                </div>

                <div>
                  <label style={{ display: 'block', fontSize: '13px', fontWeight: '600', color: '#334155', marginBottom: '6px' }}>
                    🏢 Groups / Divisi
                  </label>
                  <input
                    type="text"
                    value={editForm.group_name}
                    onChange={(e) => setEditForm({ ...editForm, group_name: e.target.value })}
                    placeholder="Contoh: IT Support, Marketing, Gudang, Operasional"
                    style={{
                      width: '100%',
                      padding: '9px 12px',
                      borderRadius: '8px',
                      border: '1px solid #cbd5e1',
                      fontSize: '14px',
                      backgroundColor: '#ffffff',
                    }}
                  />
                  <span style={{ fontSize: '11px', color: '#94a3b8', marginTop: '3px', display: 'block' }}>
                    Divisi atau unit kerja pemohon di dalam perusahaan.
                  </span>
                </div>
              </div>

              {/* Row 2: Manager & Notes */}
              <div>
                <label style={{ display: 'block', fontSize: '13px', fontWeight: '600', color: '#334155', marginBottom: '6px' }}>
                  📝 Catatan / Alasan Peminjaman
                </label>
                <textarea
                  rows={2}
                  value={editForm.notes}
                  onChange={(e) => setEditForm({ ...editForm, notes: e.target.value })}
                  placeholder="Catatan tambahan keperluan peminjaman aset..."
                  style={{
                    width: '100%',
                    padding: '9px 12px',
                    borderRadius: '8px',
                    border: '1px solid #cbd5e1',
                    fontSize: '13px',
                    fontFamily: 'inherit',
                  }}
                />
              </div>

              {/* Items Section */}
              <div style={{ backgroundColor: '#f8fafc', border: '1px solid #e2e8f0', borderRadius: '12px', padding: '16px' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '12px' }}>
                  <div>
                    <h4 style={{ fontSize: '14px', fontWeight: '700', color: '#0f172a', margin: 0 }}>
                      📦 Rincian Aset yang Dipinjam ({editForm.items.length})
                    </h4>
                    <span style={{ fontSize: '12px', color: '#64748b' }}>
                      Atur durasi peminjaman (hari) atau tambah aset pendukung lainnya.
                    </span>
                  </div>

                  {!showAddAssetRow && (
                    <button
                      type="button"
                      onClick={() => setShowAddAssetRow(true)}
                      className="btn btn-outline btn-sm"
                      style={{ fontSize: '12px', display: 'flex', alignItems: 'center', gap: '4px', borderColor: '#cbd5e1', color: '#2563eb' }}
                    >
                      <i className="ph ph-plus" /> Tambah Aset
                    </button>
                  )}
                </div>

                {/* Add Asset Row (Collapsible) */}
                {showAddAssetRow && (
                  <div
                    style={{
                      backgroundColor: '#ffffff',
                      border: '1px dashed #93c5fd',
                      borderRadius: '8px',
                      padding: '12px',
                      marginBottom: '12px',
                    }}
                  >
                    <div style={{ fontWeight: '600', fontSize: '12px', color: '#1d4ed8', marginBottom: '8px' }}>
                      ➕ Tambah Aset ke Permohonan
                    </div>

                    <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap', alignItems: 'center', marginBottom: '8px' }}>
                      <input
                        type="text"
                        placeholder="Cari nama / kode aset..."
                        value={assetSearchQuery}
                        onChange={(e) => setAssetSearchQuery(e.target.value)}
                        style={{
                          width: '180px',
                          padding: '6px 10px',
                          fontSize: '12px',
                          borderRadius: '6px',
                          border: '1px solid #cbd5e1',
                        }}
                      />

                      <select
                        value={selectedNewAssetId}
                        onChange={(e) => setSelectedNewAssetId(e.target.value)}
                        style={{
                          flex: 1,
                          minWidth: '220px',
                          padding: '6px 10px',
                          fontSize: '12px',
                          borderRadius: '6px',
                          border: '1px solid #cbd5e1',
                        }}
                      >
                        <option value="">-- Pilih Aset dari Katalog --</option>
                        {filteredAssetOptions.map((a) => (
                          <option key={a.asset_id} value={a.asset_id}>
                            {a.name} ({a.asset_id}) {a.category ? `[${a.category}]` : ''}
                          </option>
                        ))}
                      </select>

                      <div style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
                        <input
                          type="number"
                          min="1"
                          value={newAssetDuration}
                          onChange={(e) => setNewAssetDuration(parseInt(e.target.value, 10) || 1)}
                          style={{
                            width: '60px',
                            padding: '6px 8px',
                            fontSize: '12px',
                            borderRadius: '6px',
                            border: '1px solid #cbd5e1',
                            textAlign: 'center',
                          }}
                        />
                        <span style={{ fontSize: '12px', color: '#64748b' }}>hari</span>
                      </div>
                    </div>

                    <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '8px' }}>
                      <button
                        type="button"
                        onClick={() => {
                          setShowAddAssetRow(false)
                          setSelectedNewAssetId('')
                        }}
                        className="btn btn-ghost btn-sm"
                        style={{ fontSize: '12px', padding: '4px 10px' }}
                      >
                        Batal
                      </button>
                      <button
                        type="button"
                        onClick={handleAddAssetToForm}
                        className="btn btn-primary btn-sm"
                        style={{ fontSize: '12px', padding: '4px 12px', backgroundColor: '#2563eb' }}
                        disabled={!selectedNewAssetId}
                      >
                        Tambahkan ke Daftar
                      </button>
                    </div>
                  </div>
                )}

                {/* Items List */}
                <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
                  {editForm.items.map((it, idx) => (
                    <div
                      key={idx}
                      style={{
                        backgroundColor: '#ffffff',
                        border: '1px solid #e2e8f0',
                        borderRadius: '8px',
                        padding: '12px 14px',
                        display: 'flex',
                        justifyContent: 'space-between',
                        alignItems: 'center',
                        flexWrap: 'wrap',
                        gap: '12px',
                      }}
                    >
                      <div style={{ minWidth: '180px', flex: 1 }}>
                        <div style={{ fontWeight: '600', fontSize: '13px', color: '#1e293b' }}>
                          {idx + 1}. {it.asset_name || it.asset_id}
                        </div>
                        <div style={{ fontSize: '11px', color: '#64748b', marginTop: '2px' }}>
                          Kode: <code style={{ backgroundColor: '#f1f5f9', padding: '2px 5px', borderRadius: '4px' }}>{it.asset_id}</code> {it.category ? `| ${it.category}` : ''}
                        </div>
                      </div>

                      {/* Duration Controls */}
                      <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                        <span style={{ fontSize: '12px', color: '#475569', fontWeight: '500' }}>Durasi:</span>
                        <div style={{ display: 'flex', alignItems: 'center', border: '1px solid #cbd5e1', borderRadius: '6px', overflow: 'hidden' }}>
                          <button
                            type="button"
                            onClick={() => handleItemDurationChange(idx, -1, true)}
                            style={{
                              backgroundColor: '#f8fafc',
                              border: 'none',
                              padding: '6px 8px',
                              cursor: 'pointer',
                              color: '#334155',
                              borderRight: '1px solid #cbd5e1',
                            }}
                            title="Kurang 1 hari"
                          >
                            <i className="ph ph-minus" style={{ fontSize: '12px' }} />
                          </button>
                          <input
                            type="number"
                            min="1"
                            value={it.duration_days}
                            onChange={(e) => handleItemDurationChange(idx, e.target.value, false)}
                            style={{
                              width: '50px',
                              padding: '5px 4px',
                              textAlign: 'center',
                              border: 'none',
                              fontSize: '13px',
                              fontWeight: '600',
                            }}
                          />
                          <button
                            type="button"
                            onClick={() => handleItemDurationChange(idx, 1, true)}
                            style={{
                              backgroundColor: '#f8fafc',
                              border: 'none',
                              padding: '6px 8px',
                              cursor: 'pointer',
                              color: '#334155',
                              borderLeft: '1px solid #cbd5e1',
                            }}
                            title="Tambah 1 hari"
                          >
                            <i className="ph ph-plus" style={{ fontSize: '12px' }} />
                          </button>
                        </div>
                        <span style={{ fontSize: '12px', color: '#64748b' }}>hari</span>

                        {/* Quick Presets */}
                        <div style={{ display: 'flex', gap: '4px', marginLeft: '4px' }}>
                          {[7, 14, 30].map((presetDays) => (
                            <button
                              key={presetDays}
                              type="button"
                              onClick={() => handleItemDurationChange(idx, presetDays, false)}
                              style={{
                                padding: '3px 7px',
                                fontSize: '11px',
                                borderRadius: '4px',
                                border: '1px solid #e2e8f0',
                                backgroundColor: it.duration_days === presetDays ? '#dbeafe' : '#f8fafc',
                                color: it.duration_days === presetDays ? '#1d4ed8' : '#64748b',
                                cursor: 'pointer',
                                fontWeight: it.duration_days === presetDays ? '700' : 'normal',
                              }}
                            >
                              {presetDays}h
                            </button>
                          ))}
                        </div>

                        {/* Remove item button */}
                        {editForm.items.length > 1 && (
                          <button
                            type="button"
                            onClick={() => handleRemoveItem(idx)}
                            style={{
                              background: 'none',
                              border: 'none',
                              color: '#ef4444',
                              cursor: 'pointer',
                              padding: '6px',
                              marginLeft: '6px',
                            }}
                            title="Hapus aset dari daftar"
                          >
                            <i className="ph ph-trash" style={{ fontSize: '16px' }} />
                          </button>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </div>

            {/* Modal Actions Footer */}
            <div
              style={{
                marginTop: '24px',
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                flexWrap: 'wrap',
                gap: '10px',
                borderTop: '1px solid #f1f5f9',
                paddingTop: '16px',
              }}
            >
              <button
                type="button"
                onClick={closeEditModal}
                disabled={isSaving || isSubmitting}
                className="btn btn-ghost btn-sm"
              >
                Batal
              </button>

              <div style={{ display: 'flex', gap: '10px' }}>
                <button
                  type="button"
                  onClick={handleSaveEdit}
                  disabled={isSaving || isSubmitting}
                  className="btn btn-outline btn-sm"
                  style={{ color: '#2563eb', borderColor: '#93c5fd' }}
                >
                  {isSaving ? (
                    <>
                      <i className="ph ph-spinner ph-spin" /> Menyimpan...
                    </>
                  ) : (
                    <>
                      <i className="ph ph-floppy-disk" /> Simpan Perubahan
                    </>
                  )}
                </button>

                <button
                  type="button"
                  onClick={handleConfirmAndSubmit}
                  disabled={isSaving || isSubmitting}
                  className="btn btn-primary btn-sm"
                  style={{
                    backgroundColor: '#16a34a',
                    borderColor: '#16a34a',
                    padding: '8px 18px',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '6px',
                  }}
                >
                  {isSubmitting ? (
                    <>
                      <i className="ph ph-spinner ph-spin" /> Memproses...
                    </>
                  ) : (
                    <>
                      <i className="ph ph-check-circle" /> Konfirmasi & Submit Resmi
                    </>
                  )}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
