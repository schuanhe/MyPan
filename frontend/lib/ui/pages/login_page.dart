import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:go_router/go_router.dart';
import 'dart:ui';
import '../../providers/auth_provider.dart';
import '../../services/api_service.dart';

class LoginPage extends StatefulWidget {
  final String? redirect; // 登录后回跳地址
  const LoginPage({super.key, this.redirect});

  @override
  State<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends State<LoginPage> {
  final _usernameCtrl = TextEditingController();
  final _passwordCtrl = TextEditingController();
  bool _isRegister = false;
  bool _loading = false;
  bool _checkingStatus = true;
  bool _systemInitialized = false;
  int _durationSeconds = 3600 * 24 * 7; // 默认 7 天

  @override
  void initState() {
    super.initState();
    _checkSystemStatus();
  }

  Future<void> _checkSystemStatus() async {
    try {
      final res = await ApiService().dio.get('/auth/status');
      if (res.statusCode == 200) {
        final initialized = res.data['initialized'] == true;
        setState(() {
          _systemInitialized = initialized;
          _isRegister = !initialized;
          _checkingStatus = false;
        });
      }
    } catch (e) {
      setState(() => _checkingStatus = false);
    }
  }

  void _submit() async {
    final user = _usernameCtrl.text;
    final pass = _passwordCtrl.text;
    if (user.isEmpty || pass.isEmpty) return;

    setState(() => _loading = true);
    final auth = context.read<AuthProvider>();
    bool success;

    if (_isRegister) {
      success = await auth.register(user, pass);
      if (success) {
        ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('注册成功，请使用新账号登录！')));
        setState(() => _isRegister = false);
      } else {
        ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('注册受阻，或许是被抢注了或者网络异常')));
      }
    } else {
      success = await auth.login(user, pass, durationSeconds: _durationSeconds);
      if (success) {
        // 处理回跳
        if (widget.redirect != null && widget.redirect!.isNotEmpty) {
          context.go(widget.redirect!);
        } else {
          context.go('/dashboard');
        }
      } else {
        ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('敲错密码了吗？请核实再试')));
      }
    }
    
    if (mounted) setState(() => _loading = false);
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Container(
        decoration: const BoxDecoration(
          gradient: LinearGradient(
            begin: Alignment.topLeft,
            end: Alignment.bottomRight,
            colors: [Color(0xFFE5E7EB), Color(0xFFF3F4F6), Color(0xFFFFFFFF)],
          ),
        ),
        child: Center(
          child: ClipRRect(
            borderRadius: BorderRadius.circular(24),
            child: BackdropFilter(
              filter: ImageFilter.blur(sigmaX: 16, sigmaY: 16),
              child: Container(
                width: 380,
                padding: const EdgeInsets.all(40),
                decoration: BoxDecoration(
                  color: Colors.white.withOpacity(0.7),
                  borderRadius: BorderRadius.circular(24),
                  border: Border.all(color: Colors.white.withOpacity(0.8), width: 1.5),
                  boxShadow: [
                    BoxShadow(color: Colors.black.withOpacity(0.05), blurRadius: 24, spreadRadius: 0),
                  ],
                ),
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    const Icon(Icons.cloud_outlined, size: 64, color: Color(0xFF4F46E5)),
                    const SizedBox(height: 16),
                    Text(
                      _checkingStatus ? '连接中...' : (_isRegister ? '加入 MyPan' : '欢迎回来'),
                      style: Theme.of(context).textTheme.headlineSmall?.copyWith(fontWeight: FontWeight.w700),
                    ),
                    const SizedBox(height: 8),
                    Text('高级个人网盘架构系统', style: Theme.of(context).textTheme.bodyMedium?.copyWith(color: Colors.grey[600])),
                    const SizedBox(height: 32),
                    TextField(
                      controller: _usernameCtrl,
                      decoration: const InputDecoration(labelText: '用户名', prefixIcon: Icon(Icons.person_outline)),
                    ),
                    const SizedBox(height: 16),
                    TextField(
                      controller: _passwordCtrl,
                      obscureText: true,
                      decoration: const InputDecoration(labelText: '密码', prefixIcon: Icon(Icons.lock_outline)),
                    ),
                    if (!_isRegister) ...[
                      const SizedBox(height: 16),
                      DropdownButtonFormField<int>(
                        value: _durationSeconds,
                        decoration: const InputDecoration(labelText: '保持登录时长', prefixIcon: Icon(Icons.timer_outlined)),
                        items: const [
                          DropdownMenuItem(value: 3600, child: Text('1 小时')),
                          DropdownMenuItem(value: 3600 * 24, child: Text('1 天')),
                          DropdownMenuItem(value: 3600 * 24 * 7, child: Text('1 周')),
                          DropdownMenuItem(value: 3600 * 24 * 365, child: Text('1 年')),
                          DropdownMenuItem(value: 2147483647, child: Text('永久')),
                        ],
                        onChanged: (val) => setState(() => _durationSeconds = val!),
                      ),
                    ],
                    const SizedBox(height: 32),
                    SizedBox(
                      width: double.infinity,
                      height: 48,
                      child: ElevatedButton(
                        style: ElevatedButton.styleFrom(
                          backgroundColor: const Color(0xFF4F46E5),
                          foregroundColor: Colors.white,
                          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
                          elevation: 0,
                        ),
                        onPressed: (_loading || _checkingStatus) ? null : _submit,
                        child: _loading
                            ? const SizedBox(width: 24, height: 24, child: CircularProgressIndicator(color: Colors.white, strokeWidth: 2))
                            : Text(_isRegister ? '挂载注册' : '登录 / 授权', style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w600)),
                      ),
                    ),
                    if (!_systemInitialized && !_checkingStatus) ...[
                      const SizedBox(height: 16),
                      TextButton(
                        onPressed: () => setState(() => _isRegister = !_isRegister),
                        child: Text(_isRegister ? '已有账号？转去登录' : '没有档案？立即开户', style: const TextStyle(color: Color(0xFF4F46E5))),
                      )
                    ]
                  ],
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}
