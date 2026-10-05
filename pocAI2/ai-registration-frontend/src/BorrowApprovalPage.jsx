import React, { useState, useEffect } from 'react'
import {
  getBorrowApprovals,
  getBorrowTransaction,
  checkBorrowAnomalies,
  submitBorrowApproval,
  createBorrowRequest,
  getAssets,
} from './services.js'

export default function BorrowApprovalPage({ showToast, onBack, onGoHold, initialTrxId }) {
  const [approvals, setApprovals] = useState([])
  const [loadingList, setLoadingList] = useState(true)
  const [selectedTrxId, setSelectedTrxId] = useState(null)
  const [trxDetail, setTrxDetail] = useState(null)
  const [loadingDetail, setLoadingDetail] = useState(false)

  // Item form states
  const [itemsState, setItemsState] = useState([])
  const [submitAttempted, setSubmitAttempted] = useState(false)
  const [formError, setFormError] = useState('')

  // Anomaly Modal State
  const [anomalyModalOpen, setAnomalyModalOpen] = useState(false)
  const [loadingAnomaly, setLoadingAnomaly] = useState(false)
  const [anomalyData, setAnomalyData] = useState(null)
  const [statusFilter, setStatusFilter] = useState('all') // 'all' | 'pending_approval' | 'approved' | 'rejected'
  const [newModalOpen, setNewModalOpen] = useState(false)
  const [availableAssets, setAvailableAssets] = useState([])
  const [submittingNew, setSubmittingNew] = useState(false)
  const [newFormError, setNewFormError] = useState('')
  const [newForm, setNewForm] = useState({
    borrower_name: '',
    group_name: 'qtera mandiri',
    items: [{ asset_id: '', duration_days: 14 }],
  })

  // Load approvals list
  const loadApprovals = async (targetId = initialTrxId) => {
    setLoadingList(true)
    try {
      const res = await getBorrowApprovals()
      const txs = res.transactions || []
      setApprovals(txs)
      if (txs.length > 0) {
        if (targetId) {
          const match = txs.find((t) => t.id === targetId || t.transaction_code === targetId)
          if (match) {
            handleSelectTrx(match.id)
          } else {
            handleSelectTrx(targetId)
          }
        } else if (!selectedTrxId) {
          // default select first
          handleSelectTrx(txs[0].id)
        }
      }
    } catch (err) {
      showToast('Gagal memuat daftar approval: ' + err.message, 'warning')
    } finally {
      setLoadingList(false)
    }
  }

  useEffect(() => {
    loadApprovals(initialTrxId)
  }, [initialTrxId])

  // Select transaction
  const handleSelectTrx = async (id) => {
    setSelectedTrxId(id)
    setLoadingDetail(true)
    setAnomalyData(null)
    setSubmitAttempted(false)
    setFormError('')
    try {
      const res = await getBorrowTransaction(id)
      setTrxDetail(res)
      setItemsState(
        (res.items || []).map((it) => ({
          id: it.id,
          asset_id: it.asset_id,
          asset_name: it.asset_name,
          brand: it.brand,
          model_type: it.model_type,
          duration_days: it.duration_days,
          approved_duration_days: it.approved_duration_days ? String(it.approved_duration_days) : '',
          note: it.note || '',
          is_approved: it.is_approved !== null && it.is_approved !== undefined ? it.is_approved : null,
        }))
      )
    } catch (err) {
      showToast('Gagal memuat detail transaksi: ' + err.message, 'warning')
    } finally {
      setLoadingDetail(false)
    }
  }

  // Trigger Anomaly Detection
  const handleCheckAnomalies = async () => {
    if (!selectedTrxId) return
    setLoadingAnomaly(true)
    setAnomalyModalOpen(true)
    try {
      const res = await checkBorrowAnomalies(selectedTrxId)
      setAnomalyData(res)
    } catch (err) {
      showToast('Gagal menjalankan deteksi anomali: ' + err.message, 'warning')
    } finally {
      setLoadingAnomaly(false)
    }
  }

  // Submit Approval
  const handleSubmitDecision = async (decision) => {
    if (!selectedTrxId) return
    setSubmitAttempted(true)
    setFormError('')

    if (decision === 'approve') {
      // Validasi tiap item: approved_duration_days wajib diisi & is_approved wajib dipilih
      for (const it of itemsState) {
        if (!it.approved_duration_days || it.approved_duration_days.trim() === '' || parseInt(it.approved_duration_days, 10) <= 0) {
          const errMsg = `Approve Duration untuk aset "${it.asset_name || it.asset_id}" wajib diisi!`
          setFormError(errMsg)
          showToast(errMsg, 'warning')
          return
        }
        if (it.is_approved === null) {
          const errMsg = `Status persetujuan untuk aset "${it.asset_name || it.asset_id}" belum dipilih (Setuju / Tolak)!`
          setFormError(errMsg)
          showToast(errMsg, 'warning')
          return
        }
      }
    }

    try {
      await submitBorrowApproval(selectedTrxId, {
        decision,
        items: itemsState.map((it) => ({
          id: it.id,
          approved_duration_days: it.approved_duration_days ? parseInt(it.approved_duration_days, 10) : null,
          note: it.note ? it.note.trim() : '',
          is_approved: decision === 'reject' ? false : it.is_approved,
        })),
      })
      if (decision === 'reject') {
        showToast(`Transaksi ${trxDetail?.transaction_code} berhasil ditolak dan dihapus dari antrean.`, 'default')
        setSelectedTrxId(null)
        setTrxDetail(null)
        setSubmitAttempted(false)
        setFormError('')
        await loadApprovals(null)
      } else {
        showToast(`Transaksi ${trxDetail?.transaction_code} berhasil disetujui.`, 'default')
        setSubmitAttempted(false)
        setFormError('')
        await loadApprovals(selectedTrxId)
        await handleSelectTrx(selectedTrxId)
      }
    } catch (err) {
      const errMsg = 'Gagal menyimpan keputusan approval: ' + err.message
      setFormError(errMsg)
      showToast(errMsg, 'warning')
    }
  }

  // Create New Request Handlers
  const openNewRequestModal = async () => {
    setNewForm({
      borrower_name: '',
      group_name: 'qtera mandiri',
      items: [{ asset_id: '', duration_days: 14 }],
    })
    setNewFormError('')
    setNewModalOpen(true)
    try {
      const res = await getAssets({ pageSize: 100 })
      const list = res.assets || []
      setAvailableAssets(list)
      if (list.length > 0) {
        setNewForm((prev) => ({
          ...prev,
          items: [{ asset_id: list[0].assetId || list[0].id, duration_days: 14 }],
        }))
      }
    } catch (e) {
      console.error(e)
    }
  }

  const handleAddItemToNew = () => {
    const defaultAssetId = availableAssets.length > 0 ? (availableAssets[0].assetId || availableAssets[0].id) : ''
    setNewForm((prev) => ({
      ...prev,
      items: [...prev.items, { asset_id: defaultAssetId, duration_days: 14 }],
    }))
  }

  const handleRemoveItemFromNew = (index) => {
    setNewForm((prev) => ({
      ...prev,
      items: prev.items.filter((_, i) => i !== index),
    }))
  }

  const handleSubmitNewRequest = async (e) => {
    e.preventDefault()
    if (!newForm.borrower_name.trim()) {
      setNewFormError('Nama peminjam wajib diisi!')
      return
    }
    for (let i = 0; i < newForm.items.length; i++) {
      const item = newForm.items[i]
      if (!item.asset_id) {
        setNewFormError(`Aset ke-${i + 1} belum dipilih!`)
        return
      }
      if (!item.duration_days || parseInt(item.duration_days, 10) <= 0) {
        setNewFormError(`Durasi pinjam aset ke-${i + 1} harus lebih dari 0 hari!`)
        return
      }
    }

    setSubmittingNew(true)
    setNewFormError('')
    try {
      const res = await createBorrowRequest({
        borrower_name: newForm.borrower_name.trim(),
        group_name: newForm.group_name.trim() || 'qtera mandiri',
        items: newForm.items.map((it) => ({
          asset_id: it.asset_id,
          duration_days: parseInt(it.duration_days, 10),
        })),
      })
      showToast(`Permohonan ${res.transaction_code} berhasil diajukan!`, 'default')
      setNewModalOpen(false)
      await loadApprovals()
      if (res.transaction_id) {
        handleSelectTrx(res.transaction_id)
      }
    } catch (err) {
      setNewFormError('Gagal mengajukan peminjaman: ' + err.message)
    } finally {
      setSubmittingNew(false)
    }
  }

  return (
    <div className="borrow-approval-container" style={{ padding: '24px', maxWidth: '1280px', margin: '0 auto' }}>
      {/* Top Bar Navigation */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '24px' }}>
        <div>
          <h1 style={{ fontSize: '24px', fontWeight: '700', color: '#1e293b', margin: 0 }}>
            Halaman Approval Peminjaman Aset (Borrow)
          </h1>
          <p style={{ color: '#64748b', fontSize: '14px', margin: '4px 0 0 0' }}>
            PoC Approval Anomaly Inspection — Evaluasi 7 Sinyal Risiko & Verifikasi Kelayakan Peminjaman
          </p>
        </div>
        <div style={{ display: 'flex', gap: '10px' }}>
          {onGoHold && (
            <button
              onClick={onGoHold}
              className="btn btn-outline btn-sm"
              style={{ display: 'flex', alignItems: 'center', gap: '6px', color: '#b45309', borderColor: '#fde68a' }}
            >
              <i className="ph ph-hourglass-high" /> Buka Hold Asset (AI)
            </button>
          )}
          <button
            onClick={openNewRequestModal}
            className="btn btn-primary btn-sm"
            style={{ display: 'flex', alignItems: 'center', gap: '6px' }}
          >
            <i className="ph ph-plus-circle" style={{ fontSize: '16px' }} /> Buat Pengajuan Pinjam Baru
          </button>
          <button
            className="btn btn-ghost btn-sm"
            onClick={onBack}
            style={{ display: 'flex', alignItems: 'center', gap: '6px' }}
          >
            <i className="ph ph-arrow-left" /> Kembali ke Asset List
          </button>
        </div>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '320px 1fr', gap: '24px' }}>
        {/* Left Sidebar: Daftar Transaksi Approval */}
        <div
          style={{
            backgroundColor: '#ffffff',
            borderRadius: '12px',
            border: '1px solid #e2e8f0',
            padding: '16px',
            boxShadow: '0 1px 3px rgba(0,0,0,0.05)',
            maxHeight: 'calc(100vh - 160px)',
            overflowY: 'auto',
          }}
        >
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '8px' }}>
            <span style={{ fontSize: '14px', fontWeight: '700', color: '#1e293b' }}>
              Daftar Approval ({approvals.length})
            </span>
            <button
              onClick={() => loadApprovals(selectedTrxId)}
              className="btn btn-ghost btn-xs"
              title="Refresh daftar"
              style={{ padding: '4px 8px' }}
            >
              <i className="ph ph-arrows-clockwise" />
            </button>
          </div>

          {/* Status Filter Tabs */}
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '4px', marginBottom: '12px', backgroundColor: '#f1f5f9', padding: '3px', borderRadius: '8px' }}>
            <button
              onClick={() => setStatusFilter('all')}
              style={{
                border: 'none',
                borderRadius: '6px',
                padding: '4px 2px',
                fontSize: '11px',
                fontWeight: statusFilter === 'all' ? '700' : '500',
                backgroundColor: statusFilter === 'all' ? '#ffffff' : 'transparent',
                color: statusFilter === 'all' ? '#1e293b' : '#64748b',
                cursor: 'pointer',
                boxShadow: statusFilter === 'all' ? '0 1px 2px rgba(0,0,0,0.06)' : 'none',
              }}
            >
              Semua ({approvals.length})
            </button>
            <button
              onClick={() => setStatusFilter('pending_approval')}
              style={{
                border: 'none',
                borderRadius: '6px',
                padding: '4px 2px',
                fontSize: '11px',
                fontWeight: statusFilter === 'pending_approval' ? '700' : '500',
                backgroundColor: statusFilter === 'pending_approval' ? '#ffffff' : 'transparent',
                color: statusFilter === 'pending_approval' ? '#b45309' : '#64748b',
                cursor: 'pointer',
                boxShadow: statusFilter === 'pending_approval' ? '0 1px 2px rgba(0,0,0,0.06)' : 'none',
              }}
            >
              Pending ({approvals.filter((t) => t.status === 'pending_approval').length})
            </button>
            <button
              onClick={() => setStatusFilter('approved')}
              style={{
                border: 'none',
                borderRadius: '6px',
                padding: '4px 2px',
                fontSize: '11px',
                fontWeight: statusFilter === 'approved' ? '700' : '500',
                backgroundColor: statusFilter === 'approved' ? '#ffffff' : 'transparent',
                color: statusFilter === 'approved' ? '#15803d' : '#64748b',
                cursor: 'pointer',
                boxShadow: statusFilter === 'approved' ? '0 1px 2px rgba(0,0,0,0.06)' : 'none',
              }}
            >
              Setuju ({approvals.filter((t) => t.status === 'approved').length})
            </button>
          </div>

          {loadingList ? (
            <div style={{ padding: '32px 0', textAlign: 'center', color: '#94a3b8' }}>
              <span className="spinner" /> Memuat daftar...
            </div>
          ) : approvals.filter((t) => statusFilter === 'all' || t.status === statusFilter).length === 0 ? (
            <div style={{ padding: '32px 16px', textAlign: 'center', color: '#94a3b8' }}>
              <i className="ph ph-check-circle" style={{ fontSize: '32px', color: '#10b981', display: 'block', marginBottom: '8px' }} />
              Tidak ada permohonan dalam kategori ini.
            </div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
              {approvals
                .filter((t) => statusFilter === 'all' || t.status === statusFilter)
                .map((trx) => {
                  const isSelected = trx.id === selectedTrxId
                  return (
                    <div
                      key={trx.id}
                      onClick={() => handleSelectTrx(trx.id)}
                      style={{
                        padding: '12px',
                        borderRadius: '8px',
                        border: isSelected ? '2px solid #6366f1' : '1px solid #e2e8f0',
                        backgroundColor: isSelected ? '#f5f3ff' : '#f8fafc',
                        cursor: 'pointer',
                        transition: 'all 0.15s ease',
                      }}
                    >
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '4px' }}>
                        <span style={{ fontWeight: '700', fontSize: '13px', color: isSelected ? '#4f46e5' : '#1e293b' }}>
                          {trx.transaction_code}
                        </span>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
                          {trx.status === 'rejected' ? (
                            <span style={{ fontSize: '10px', fontWeight: '700', padding: '1px 5px', borderRadius: '4px', backgroundColor: '#fee2e2', color: '#b91c1c' }}>
                              Ditolak
                            </span>
                          ) : trx.status === 'approved' ? (
                            <span style={{ fontSize: '10px', fontWeight: '700', padding: '1px 5px', borderRadius: '4px', backgroundColor: '#dcfce7', color: '#15803d' }}>
                              Disetujui
                            </span>
                          ) : (
                            <span style={{ fontSize: '10px', fontWeight: '700', padding: '1px 5px', borderRadius: '4px', backgroundColor: '#fef3c7', color: '#92400e' }}>
                              Pending
                            </span>
                          )}
                          <span
                            style={{
                              fontSize: '11px',
                              fontWeight: '600',
                              padding: '1px 5px',
                              borderRadius: '4px',
                              backgroundColor: '#e0e7ff',
                              color: '#3730a3',
                            }}
                          >
                            {trx.item_count || 1} aset
                          </span>
                        </div>
                      </div>
                      <div style={{ fontSize: '12px', color: '#475569', display: 'flex', flexDirection: 'column', gap: '2px' }}>
                        <div><strong>Peminjam:</strong> {trx.borrower_name}</div>
                        <div><strong>Group:</strong> {trx.group_name}</div>
                        <div style={{ color: '#94a3b8', fontSize: '11px', marginTop: '4px' }}>
                          {new Date(trx.request_date).toLocaleDateString('id-ID', { day: 'numeric', month: 'short', year: 'numeric' })}
                        </div>
                      </div>
                    </div>
                  )
                })}
            </div>
          )}
        </div>

        {/* Right Content: Header Transaksi, Check Anomalies Button, & Asset Table */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
          {loadingDetail ? (
            <div
              style={{
                backgroundColor: '#ffffff',
                borderRadius: '12px',
                border: '1px solid #e2e8f0',
                padding: '48px',
                textAlign: 'center',
                color: '#64748b',
              }}
            >
              <span className="spinner" /> Memuat rincian transaksi...
            </div>
          ) : !trxDetail ? (
            <div
              style={{
                backgroundColor: '#ffffff',
                borderRadius: '12px',
                border: '1px solid #e2e8f0',
                padding: '48px',
                textAlign: 'center',
                color: '#94a3b8',
              }}
            >
              <i className="ph ph-hand-pointing" style={{ fontSize: '36px', marginBottom: '12px', display: 'block' }} />
              Pilih salah satu transaksi peminjaman di sebelah kiri untuk melihat rincian approval.
            </div>
          ) : (
            <>
              {/* 1. Header Halaman Approval (Per Transaksi) */}
              <div
                style={{
                  backgroundColor: '#ffffff',
                  borderRadius: '12px',
                  border: '1px solid #e2e8f0',
                  padding: '20px',
                  boxShadow: '0 1px 3px rgba(0,0,0,0.05)',
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center',
                  flexWrap: 'wrap',
                  gap: '16px',
                }}
              >
                <div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '6px' }}>
                    <span style={{ fontSize: '18px', fontWeight: '800', color: '#1e293b' }}>
                      {trxDetail.transaction_code}
                    </span>
                    {trxDetail.status === 'rejected' ? (
                      <span
                        style={{
                          fontSize: '12px',
                          padding: '3px 8px',
                          borderRadius: '12px',
                          backgroundColor: '#fee2e2',
                          color: '#b91c1c',
                          fontWeight: '700',
                        }}
                      >
                        🔴 Ditolak
                      </span>
                    ) : trxDetail.status === 'approved' ? (
                      <span
                        style={{
                          fontSize: '12px',
                          padding: '3px 8px',
                          borderRadius: '12px',
                          backgroundColor: '#dcfce7',
                          color: '#15803d',
                          fontWeight: '700',
                        }}
                      >
                        🟢 Disetujui
                      </span>
                    ) : trxDetail.status === 'active' ? (
                      <span
                        style={{
                          fontSize: '12px',
                          padding: '3px 8px',
                          borderRadius: '12px',
                          backgroundColor: '#e0e7ff',
                          color: '#3730a3',
                          fontWeight: '700',
                        }}
                      >
                        🔵 Sedang Dipinjam
                      </span>
                    ) : (
                      <span
                        style={{
                          fontSize: '12px',
                          padding: '3px 8px',
                          borderRadius: '12px',
                          backgroundColor: '#fef3c7',
                          color: '#92400e',
                          fontWeight: '700',
                        }}
                      >
                        🟡 Menunggu Approval
                      </span>
                    )}
                  </div>
                  <div style={{ display: 'flex', gap: '20px', fontSize: '13px', color: '#475569', flexWrap: 'wrap' }}>
                    <div>
                      <span style={{ color: '#94a3b8' }}>Peminjam:</span>{' '}
                      <strong>{trxDetail.borrower_name}</strong>
                    </div>
                    <div>
                      <span style={{ color: '#94a3b8' }}>Manager Request:</span>{' '}
                      <strong>{trxDetail.manager_name || 'System / Direct'}</strong>
                    </div>
                    <div>
                      <span style={{ color: '#94a3b8' }}>Group Asset:</span>{' '}
                      <strong>{trxDetail.group_name}</strong>
                    </div>
                  </div>
                </div>

                {/* Tombol Check Anomalies (Khusus Halaman Approval) */}
                <div>
                  <button
                    onClick={handleCheckAnomalies}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: '8px',
                      padding: '10px 18px',
                      background: 'linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%)',
                      color: '#ffffff',
                      border: 'none',
                      borderRadius: '8px',
                      fontSize: '14px',
                      fontWeight: '700',
                      cursor: 'pointer',
                      boxShadow: '0 4px 12px rgba(99, 102, 241, 0.3)',
                      transition: 'all 0.2s ease',
                    }}
                    title="Jalankan inspeksi 7 sinyal anomali terhadap transaksi ini"
                  >
                    <i className="ph ph-shield-warning" style={{ fontSize: '18px' }} />
                    Check Anomalies
                  </button>
                </div>
              </div>

              {/* 2. Tabel Aset di dalam Transaksi */}
              <div
                style={{
                  backgroundColor: '#ffffff',
                  borderRadius: '12px',
                  border: '1px solid #e2e8f0',
                  boxShadow: '0 1px 3px rgba(0,0,0,0.05)',
                  overflow: 'hidden',
                }}
              >
                <div
                  style={{
                    padding: '16px 20px',
                    borderBottom: '1px solid #e2e8f0',
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                  }}
                >
                  <h3 style={{ margin: 0, fontSize: '15px', fontWeight: '700', color: '#1e293b' }}>
                    Daftar Aset yang Diajukan ({itemsState.length} item)
                  </h3>
                  <span style={{ fontSize: '12px', color: '#64748b' }}>
                    Tentukan Approve Duration & Status Persetujuan per Item
                  </span>
                </div>

                <div style={{ overflowX: 'auto' }}>
                  <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px' }}>
                    <thead>
                      <tr style={{ backgroundColor: '#f8fafc', borderBottom: '1px solid #e2e8f0', color: '#64748b', textAlign: 'left' }}>
                        <th style={{ padding: '12px 16px' }}>Asset Name</th>
                        <th style={{ padding: '12px 16px' }}>Brand</th>
                        <th style={{ padding: '12px 16px' }}>Model / Type</th>
                        <th style={{ padding: '12px 16px' }}>Duration (Req)</th>
                        <th style={{ padding: '12px 16px' }}>Approve Duration</th>
                        <th style={{ padding: '12px 16px' }}>Note</th>
                        <th style={{ padding: '12px 16px', textAlign: 'center' }}>Switch (Approve)</th>
                      </tr>
                    </thead>
                    <tbody>
                      {itemsState.map((item, idx) => {
                        const durationEmpty = !item.approved_duration_days || item.approved_duration_days.trim() === '' || parseInt(item.approved_duration_days, 10) <= 0
                        const statusEmpty = item.is_approved === null
                        return (
                          <tr
                            key={item.id}
                            style={{
                              borderBottom: '1px solid #f1f5f9',
                              backgroundColor: item.is_approved === true ? '#f0fdf4' : item.is_approved === false ? '#fef2f2' : '#ffffff',
                              transition: 'background-color 0.15s ease',
                            }}
                          >
                            <td style={{ padding: '12px 16px', fontWeight: '600', color: '#1e293b' }}>
                              {item.asset_name || item.asset_id}
                            </td>
                            <td style={{ padding: '12px 16px', color: '#475569' }}>{item.brand || '-'}</td>
                            <td style={{ padding: '12px 16px', color: '#475569' }}>{item.model_type || '-'}</td>
                            <td style={{ padding: '12px 16px', color: '#1e293b' }}>
                              <span style={{ fontWeight: '600' }}>{item.duration_days} hari</span>
                            </td>
                            <td style={{ padding: '12px 16px' }}>
                              <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                                <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                                  <input
                                    type="number"
                                    min="1"
                                    value={item.approved_duration_days}
                                    onChange={(e) => {
                                      const val = e.target.value
                                      setItemsState((prev) =>
                                        prev.map((it, i) => (i === idx ? { ...it, approved_duration_days: val } : it))
                                      )
                                      setFormError('')
                                    }}
                                    style={{
                                      width: '70px',
                                      padding: '6px 8px',
                                      borderRadius: '6px',
                                      border: submitAttempted && durationEmpty ? '1px solid #ef4444' : '1px solid #cbd5e1',
                                      backgroundColor: submitAttempted && durationEmpty ? '#fef2f2' : '#ffffff',
                                      fontSize: '13px',
                                    }}
                                  />
                                  <span style={{ fontSize: '11px', color: '#64748b' }}>hari</span>
                                </div>
                                {submitAttempted && durationEmpty && (
                                  <span style={{ fontSize: '11px', color: '#ef4444', fontWeight: '600' }}>
                                    Wajib diisi
                                  </span>
                                )}
                              </div>
                            </td>
                            <td style={{ padding: '12px 16px' }}>
                              <input
                                type="text"
                                value={item.note}
                                onChange={(e) => {
                                  const val = e.target.value
                                  setItemsState((prev) =>
                                    prev.map((it, i) => (i === idx ? { ...it, note: val } : it))
                                  )
                                }}
                                style={{
                                  width: '100%',
                                  minWidth: '150px',
                                  padding: '6px 8px',
                                  borderRadius: '6px',
                                  border: '1px solid #cbd5e1',
                                  fontSize: '13px',
                                }}
                              />
                            </td>
                            <td style={{ padding: '12px 16px', textAlign: 'center' }}>
                              <div style={{ display: 'inline-flex', flexDirection: 'column', alignItems: 'center', gap: '4px' }}>
                                <div
                                  style={{
                                    display: 'inline-flex',
                                    borderRadius: '6px',
                                    border: submitAttempted && statusEmpty ? '1px solid #ef4444' : '1px solid #cbd5e1',
                                    overflow: 'hidden',
                                  }}
                                >
                                  <button
                                    type="button"
                                    onClick={() => {
                                      setItemsState((prev) =>
                                        prev.map((it, i) => (i === idx ? { ...it, is_approved: true } : it))
                                      )
                                      setFormError('')
                                    }}
                                    style={{
                                      padding: '5px 12px',
                                      fontSize: '12px',
                                      fontWeight: '600',
                                      border: 'none',
                                      cursor: 'pointer',
                                      backgroundColor: item.is_approved === true ? '#10b981' : '#f8fafc',
                                      color: item.is_approved === true ? '#ffffff' : '#64748b',
                                      transition: 'all 0.15s ease',
                                    }}
                                  >
                                    <i className="ph ph-check" style={{ marginRight: '4px' }} />
                                    Setuju
                                  </button>
                                  <button
                                    type="button"
                                    onClick={() => {
                                      setItemsState((prev) =>
                                        prev.map((it, i) => (i === idx ? { ...it, is_approved: false } : it))
                                      )
                                      setFormError('')
                                    }}
                                    style={{
                                      padding: '5px 12px',
                                      fontSize: '12px',
                                      fontWeight: '600',
                                      border: 'none',
                                      borderLeft: '1px solid #e2e8f0',
                                      cursor: 'pointer',
                                      backgroundColor: item.is_approved === false ? '#ef4444' : '#f8fafc',
                                      color: item.is_approved === false ? '#ffffff' : '#64748b',
                                      transition: 'all 0.15s ease',
                                    }}
                                  >
                                    <i className="ph ph-x" style={{ marginRight: '4px' }} />
                                    Tolak
                                  </button>
                                </div>
                                {submitAttempted && statusEmpty && (
                                  <span style={{ fontSize: '11px', color: '#ef4444', fontWeight: '600' }}>
                                    Pilih status
                                  </span>
                                )}
                              </div>
                            </td>
                          </tr>
                        )
                      })}
                    </tbody>
                  </table>
                </div>

                {/* Validation Error Banner */}
                {formError && (
                  <div
                    style={{
                      padding: '10px 20px',
                      backgroundColor: '#fef2f2',
                      borderTop: '1px solid #fecaca',
                      color: '#dc2626',
                      fontSize: '13px',
                      fontWeight: '600',
                      display: 'flex',
                      alignItems: 'center',
                      gap: '8px',
                    }}
                  >
                    <i className="ph ph-warning-circle" style={{ fontSize: '18px' }} />
                    {formError}
                  </div>
                )}

                {/* Submit Decision Footer */}
                {trxDetail.status === 'pending_approval' ? (
                  <div
                    style={{
                      padding: '16px 20px',
                      backgroundColor: '#f8fafc',
                      borderTop: '1px solid #e2e8f0',
                      display: 'flex',
                      justifyContent: 'flex-end',
                      gap: '12px',
                    }}
                  >
                    <button
                      onClick={() => handleSubmitDecision('reject')}
                      className="btn btn-danger btn-sm"
                      style={{ display: 'flex', alignItems: 'center', gap: '6px' }}
                    >
                      <i className="ph ph-x-circle" /> Tolak Seluruh Transaksi
                    </button>
                    <button
                      onClick={() => handleSubmitDecision('approve')}
                      className="btn btn-primary btn-sm"
                      style={{ display: 'flex', alignItems: 'center', gap: '6px' }}
                    >
                      <i className="ph ph-check-circle" /> Setujui Peminjaman
                    </button>
                  </div>
                ) : (
                  <div
                    style={{
                      padding: '16px 20px',
                      backgroundColor: trxDetail.status === 'rejected' ? '#fef2f2' : '#f0fdf4',
                      borderTop: `1px solid ${trxDetail.status === 'rejected' ? '#fecaca' : '#bbf7d0'}`,
                      display: 'flex',
                      alignItems: 'center',
                      gap: '12px',
                    }}
                  >
                    <i
                      className={trxDetail.status === 'rejected' ? 'ph ph-x-circle' : 'ph ph-check-circle'}
                      style={{
                        fontSize: '24px',
                        color: trxDetail.status === 'rejected' ? '#dc2626' : '#16a34a',
                        flexShrink: 0,
                      }}
                    />
                    <div>
                      <strong style={{ fontSize: '14px', color: trxDetail.status === 'rejected' ? '#991b1b' : '#166534' }}>
                        {trxDetail.status === 'rejected' ? 'Transaksi Telah Ditolak' : 'Transaksi Telah Disetujui'}
                      </strong>
                      <p style={{ margin: '2px 0 0', fontSize: '12px', color: trxDetail.status === 'rejected' ? '#b91c1c' : '#15803d' }}>
                        {trxDetail.status === 'rejected'
                          ? 'Data transaksi peminjaman ini tidak dihapus dan tetap tersimpan utuh dalam arsip riwayat sistem.'
                          : 'Data transaksi peminjaman ini telah resmi diproses dan tersimpan dalam sistem.'}
                      </p>
                    </div>
                  </div>
                )}
              </div>
            </>
          )}
        </div>
      </div>

      {/* 3. Popup / Modal "Check Anomalies" */}
      {anomalyModalOpen && (
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
            alignItems: 'center',
            justifyContent: 'center',
            zIndex: 9999,
            padding: '20px',
          }}
          onClick={() => setAnomalyModalOpen(false)}
        >
          <div
            style={{
              backgroundColor: '#ffffff',
              borderRadius: '16px',
              maxWidth: '760px',
              width: '100%',
              maxHeight: '90vh',
              overflowY: 'auto',
              boxShadow: '0 25px 50px -12px rgba(0, 0, 0, 0.25)',
              border: '1px solid #e2e8f0',
              padding: '24px',
            }}
            onClick={(e) => e.stopPropagation()}
          >
            {/* Modal Header */}
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '16px' }}>
              <div>
                <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                  <i className="ph ph-shield-check" style={{ fontSize: '22px', color: '#6366f1' }} />
                  <h2 style={{ fontSize: '18px', fontWeight: '800', margin: 0, color: '#0f172a' }}>
                    Hasil Pemeriksaan Anomali Peminjaman
                  </h2>
                </div>
                {anomalyData && (
                  <span style={{ fontSize: '12px', color: '#64748b', marginTop: '4px', display: 'inline-block' }}>
                    Data as of: <strong>{anomalyData.data_as_of}</strong>
                  </span>
                )}
              </div>
              <button
                onClick={() => setAnomalyModalOpen(false)}
                style={{
                  background: 'transparent',
                  border: 'none',
                  fontSize: '20px',
                  color: '#94a3b8',
                  cursor: 'pointer',
                  padding: '4px',
                }}
              >
                <i className="ph ph-x" />
              </button>
            </div>

            {loadingAnomaly ? (
              <div style={{ padding: '60px 0', textAlign: 'center', color: '#6366f1' }}>
                <span className="spinner" style={{ width: '32px', height: '32px', borderWidth: '3px' }} />
                <div style={{ marginTop: '12px', fontWeight: '600', color: '#475569' }}>
                  Mengevaluasi 7 Sinyal Anomali & Riwayat Database...
                </div>
              </div>
            ) : !anomalyData ? (
              <div style={{ padding: '40px 0', textAlign: 'center', color: '#ef4444' }}>
                Gagal memuat data anomali.
              </div>
            ) : (
              <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
                {/* Kesimpulan Section */}
                <div
                  style={{
                    borderRadius: '12px',
                    padding: '16px 20px',
                    backgroundColor: anomalyData.signals_count === 0 ? '#ecfdf5' : '#fffbeb',
                    border: anomalyData.signals_count === 0 ? '1px solid #a7f3d0' : '1px solid #fde68a',
                  }}
                >
                  <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '8px' }}>
                    {anomalyData.review_recommended ? (
                      <span
                        style={{
                          backgroundColor: '#f59e0b',
                          color: '#ffffff',
                          fontWeight: '800',
                          fontSize: '12px',
                          padding: '4px 10px',
                          borderRadius: '20px',
                          display: 'inline-flex',
                          alignItems: 'center',
                          gap: '5px',
                        }}
                      >
                        <i className="ph ph-warning-circle" /> Review Recommended
                      </span>
                    ) : (
                      <span
                        style={{
                          backgroundColor: '#10b981',
                          color: '#ffffff',
                          fontWeight: '800',
                          fontSize: '12px',
                          padding: '4px 10px',
                          borderRadius: '20px',
                          display: 'inline-flex',
                          alignItems: 'center',
                          gap: '5px',
                        }}
                      >
                        <i className="ph ph-check-circle" /> Normal (0 Sinyal)
                      </span>
                    )}

                    <span style={{ fontSize: '13px', fontWeight: '600', color: '#475569' }}>
                      {anomalyData.signals_count} sinyal anomali terdeteksi
                    </span>
                  </div>

                  {/* Narasi AI / Template / Pesan Error */}
                  {anomalyData.error_message ? (
                    <div
                      style={{
                        backgroundColor: '#fef2f2',
                        borderRadius: '8px',
                        padding: '12px 14px',
                        border: '1px solid #fecaca',
                        marginTop: '10px',
                        color: '#991b1b',
                      }}
                    >
                      <div
                        style={{
                          display: 'flex',
                          alignItems: 'center',
                          gap: '6px',
                          fontSize: '11px',
                          fontWeight: '700',
                          marginBottom: '4px',
                          textTransform: 'uppercase',
                        }}
                      >
                        <i className="ph ph-warning-circle" /> Pesan Error AI
                      </div>
                      <p style={{ margin: 0, fontSize: '13px', lineHeight: '1.5' }}>
                        {anomalyData.error_message}
                      </p>
                    </div>
                  ) : anomalyData.is_ai_generated && anomalyData.narrative ? (
                    <div
                      style={{
                        backgroundColor: '#ffffff',
                        borderRadius: '8px',
                        padding: '12px 14px',
                        border: '1px solid #fef3c7',
                        marginTop: '10px',
                      }}
                    >
                      <div
                        style={{
                          display: 'flex',
                          alignItems: 'center',
                          gap: '6px',
                          fontSize: '11px',
                          fontWeight: '700',
                          color: '#7c3aed',
                          marginBottom: '4px',
                          textTransform: 'uppercase',
                        }}
                      >
                        <i className="ph ph-sparkle" /> Rangkuman Narasi AI Assistant
                      </div>
                      <p style={{ margin: 0, fontSize: '13.5px', lineHeight: '1.5', color: '#1e293b' }}>
                        "{anomalyData.narrative}"
                      </p>
                    </div>
                  ) : null}
                </div>

                {/* Bagian 1: Konteks Peminjam */}
                <div>
                  <h4 style={{ fontSize: '13px', fontWeight: '700', textTransform: 'uppercase', color: '#64748b', marginBottom: '10px' }}>
                    Konteks Peminjam ({anomalyData.borrower_name})
                  </h4>
                  <div
                    style={{
                      border: '1px solid #e2e8f0',
                      borderRadius: '8px',
                      overflow: 'hidden',
                    }}
                  >
                    <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px' }}>
                      <thead>
                        <tr style={{ backgroundColor: '#f8fafc', borderBottom: '1px solid #e2e8f0', color: '#64748b' }}>
                          <th style={{ padding: '8px 12px', textAlign: 'left' }}>Metrik</th>
                          <th style={{ padding: '8px 12px', textAlign: 'center' }}>Di Group Ini</th>
                          <th style={{ padding: '8px 12px', textAlign: 'center' }}>Lintas Group</th>
                          <th style={{ padding: '8px 12px', textAlign: 'center' }}>Status</th>
                        </tr>
                      </thead>
                      <tbody>
                        {anomalyData.borrower_context?.map((ctxItem, i) => (
                          <tr key={i} style={{ borderBottom: '1px solid #f1f5f9' }}>
                            <td style={{ padding: '10px 12px', fontWeight: '500', color: '#1e293b' }}>{ctxItem.metric}</td>
                            <td style={{ padding: '10px 12px', textAlign: 'center', fontWeight: '600' }}>{ctxItem.group_val}</td>
                            <td style={{ padding: '10px 12px', textAlign: 'center', fontWeight: '600' }}>{ctxItem.total_val}</td>
                            <td style={{ padding: '10px 12px', textAlign: 'center' }}>
                              <span
                                style={{
                                  fontSize: '11px',
                                  fontWeight: '700',
                                  padding: '2px 8px',
                                  borderRadius: '12px',
                                  backgroundColor: ctxItem.status === 'menonjol' ? '#fee2e2' : '#dcfce7',
                                  color: ctxItem.status === 'menonjol' ? '#dc2626' : '#15803d',
                                }}
                              >
                                {ctxItem.status}
                              </span>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>

                {/* Bagian 2: Konteks Request (Evaluasi 7 Sinyal) */}
                <div>
                  <h4 style={{ fontSize: '13px', fontWeight: '700', textTransform: 'uppercase', color: '#64748b', marginBottom: '10px' }}>
                    Evaluasi Detail Sinyal Request
                  </h4>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                    {anomalyData.request_context?.map((sig) => (
                      <div
                        key={sig.id}
                        style={{
                          display: 'flex',
                          alignItems: 'flex-start',
                          gap: '12px',
                          padding: '10px 14px',
                          borderRadius: '8px',
                          backgroundColor: sig.triggered ? '#fef2f2' : '#f8fafc',
                          border: sig.triggered ? '1px solid #fecaca' : '1px solid #e2e8f0',
                        }}
                      >
                        <i
                          className={sig.triggered ? 'ph ph-warning-diamond' : 'ph ph-check-circle'}
                          style={{
                            fontSize: '18px',
                            color: sig.triggered ? '#ef4444' : '#10b981',
                            marginTop: '2px',
                          }}
                        />
                        <div style={{ flex: 1 }}>
                          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                            <span style={{ fontWeight: '700', fontSize: '13px', color: sig.triggered ? '#991b1b' : '#334155' }}>
                              Sinyal {sig.id}: {sig.name}
                            </span>
                            <span
                              style={{
                                fontSize: '11px',
                                fontWeight: '700',
                                color: sig.triggered ? '#dc2626' : '#16a34a',
                              }}
                            >
                              {sig.triggered ? 'MENONJOL' : 'NORMAL'}
                            </span>
                          </div>
                          <p style={{ margin: '3px 0 0 0', fontSize: '12px', color: sig.triggered ? '#7f1d1d' : '#64748b' }}>
                            {sig.description}
                          </p>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            )}

            {/* Modal Footer */}
            <div style={{ marginTop: '24px', display: 'flex', justifyContent: 'flex-end' }}>
              <button
                className="btn btn-ghost btn-sm"
                onClick={() => setAnomalyModalOpen(false)}
                style={{ padding: '8px 18px', borderRadius: '8px' }}
              >
                Tutup Pemeriksaan
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 4. Popup / Modal "Buat Pengajuan Pinjam Baru" */}
      {newModalOpen && (
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
            alignItems: 'center',
            justifyContent: 'center',
            zIndex: 9999,
            padding: '20px',
          }}
          onClick={() => !submittingNew && setNewModalOpen(false)}
        >
          <div
            style={{
              backgroundColor: '#ffffff',
              borderRadius: '16px',
              maxWidth: '620px',
              width: '100%',
              maxHeight: '90vh',
              overflowY: 'auto',
              boxShadow: '0 25px 50px -12px rgba(0, 0, 0, 0.25)',
              border: '1px solid #e2e8f0',
              padding: '24px',
            }}
            onClick={(e) => e.stopPropagation()}
          >
            {/* Modal Header */}
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                <i className="ph ph-hand-coins" style={{ fontSize: '22px', color: '#4f46e5' }} />
                <h2 style={{ fontSize: '18px', fontWeight: '800', margin: 0, color: '#0f172a' }}>
                  Buat Pengajuan Peminjaman Aset
                </h2>
              </div>
              <button
                type="button"
                onClick={() => setNewModalOpen(false)}
                disabled={submittingNew}
                style={{
                  background: 'transparent',
                  border: 'none',
                  fontSize: '20px',
                  color: '#94a3b8',
                  cursor: 'pointer',
                  padding: '4px',
                }}
              >
                <i className="ph ph-x" />
              </button>
            </div>

            {newFormError && (
              <div
                style={{
                  padding: '10px 14px',
                  backgroundColor: '#fef2f2',
                  border: '1px solid #fecaca',
                  borderRadius: '8px',
                  color: '#dc2626',
                  fontSize: '13px',
                  marginBottom: '16px',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '8px',
                }}
              >
                <i className="ph ph-warning-circle" style={{ fontSize: '18px' }} />
                {newFormError}
              </div>
            )}

            <form onSubmit={handleSubmitNewRequest} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
              <div>
                <label style={{ display: 'block', fontSize: '13px', fontWeight: '600', color: '#334155', marginBottom: '6px' }}>
                  Nama Peminjam
                </label>
                <input
                  type="text"
                  required
                  value={newForm.borrower_name}
                  onChange={(e) => setNewForm({ ...newForm, borrower_name: e.target.value })}
                  style={{
                    width: '100%',
                    padding: '8px 12px',
                    borderRadius: '8px',
                    border: '1px solid #cbd5e1',
                    fontSize: '14px',
                    boxSizing: 'border-box',
                  }}
                />
              </div>

              <div>
                <label style={{ display: 'block', fontSize: '13px', fontWeight: '600', color: '#334155', marginBottom: '6px' }}>
                  Nama Group / Divisi
                </label>
                <input
                  type="text"
                  value={newForm.group_name}
                  onChange={(e) => setNewForm({ ...newForm, group_name: e.target.value })}
                  style={{
                    width: '100%',
                    padding: '8px 12px',
                    borderRadius: '8px',
                    border: '1px solid #cbd5e1',
                    fontSize: '14px',
                    boxSizing: 'border-box',
                  }}
                />
              </div>

              <div>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '8px' }}>
                  <label style={{ fontSize: '13px', fontWeight: '600', color: '#334155' }}>
                    Daftar Aset yang Dipinjam ({newForm.items.length})
                  </label>
                  <button
                    type="button"
                    onClick={handleAddItemToNew}
                    style={{
                      background: 'none',
                      border: 'none',
                      color: '#4f46e5',
                      fontSize: '12px',
                      fontWeight: '700',
                      cursor: 'pointer',
                      display: 'flex',
                      alignItems: 'center',
                      gap: '4px',
                    }}
                  >
                    <i className="ph ph-plus" /> Tambah Aset Lain
                  </button>
                </div>

                <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
                  {newForm.items.map((item, idx) => (
                    <div
                      key={idx}
                      style={{
                        display: 'flex',
                        gap: '10px',
                        alignItems: 'center',
                        padding: '10px',
                        backgroundColor: '#f8fafc',
                        borderRadius: '8px',
                        border: '1px solid #e2e8f0',
                      }}
                    >
                      <div style={{ flex: 1 }}>
                        <select
                          value={item.asset_id}
                          onChange={(e) => {
                            const val = e.target.value
                            setNewForm((prev) => ({
                              ...prev,
                              items: prev.items.map((it, i) => (i === idx ? { ...it, asset_id: val } : it)),
                            }))
                          }}
                          style={{
                            width: '100%',
                            padding: '8px 10px',
                            borderRadius: '6px',
                            border: '1px solid #cbd5e1',
                            fontSize: '13px',
                            backgroundColor: '#ffffff',
                          }}
                        >
                          {availableAssets.length === 0 && <option value="">Memuat aset...</option>}
                          {availableAssets.map((ast) => (
                            <option key={ast.assetId || ast.id} value={ast.assetId || ast.id}>
                              {ast.name} ({ast.category || 'Asset'} - {ast.assetId || ast.id})
                            </option>
                          ))}
                        </select>
                      </div>

                      <div style={{ width: '110px', display: 'flex', alignItems: 'center', gap: '4px' }}>
                        <input
                          type="number"
                          min="1"
                          required
                          value={item.duration_days}
                          onChange={(e) => {
                            const val = e.target.value
                            setNewForm((prev) => ({
                              ...prev,
                              items: prev.items.map((it, i) => (i === idx ? { ...it, duration_days: val } : it)),
                            }))
                          }}
                          style={{
                            width: '65px',
                            padding: '8px',
                            borderRadius: '6px',
                            border: '1px solid #cbd5e1',
                            fontSize: '13px',
                          }}
                        />
                        <span style={{ fontSize: '12px', color: '#64748b' }}>hari</span>
                      </div>

                      {newForm.items.length > 1 && (
                        <button
                          type="button"
                          onClick={() => handleRemoveItemFromNew(idx)}
                          style={{
                            background: 'none',
                            border: 'none',
                            color: '#ef4444',
                            cursor: 'pointer',
                            padding: '6px',
                          }}
                          title="Hapus aset ini"
                        >
                          <i className="ph ph-trash" style={{ fontSize: '16px' }} />
                        </button>
                      )}
                    </div>
                  ))}
                </div>
              </div>

              {/* Action Buttons */}
              <div style={{ marginTop: '16px', display: 'flex', justifyContent: 'flex-end', gap: '10px' }}>
                <button
                  type="button"
                  onClick={() => setNewModalOpen(false)}
                  disabled={submittingNew}
                  className="btn btn-ghost btn-sm"
                  style={{ padding: '8px 16px' }}
                >
                  Batal
                </button>
                <button
                  type="submit"
                  disabled={submittingNew}
                  className="btn btn-primary btn-sm"
                  style={{ display: 'flex', alignItems: 'center', gap: '6px', padding: '8px 18px' }}
                >
                  {submittingNew ? (
                    <><span className="spinner" /> Mengirim...</>
                  ) : (
                    <><i className="ph ph-paper-plane-tilt" /> Kirim Pengajuan</>
                  )}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
